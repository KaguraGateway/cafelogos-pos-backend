package application

import (
	"context"

	"github.com/KaguraGateway/cafelogos-pos-backend/domain/model"
	"github.com/KaguraGateway/cafelogos-pos-backend/domain/repository"
	"github.com/cockroachdb/errors"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type UpdateProduct interface {
	Execute(id string, param *ProductParam) error
}

type updateProductUseCase struct {
	ctx                 context.Context
	productRepo         repository.ProductRepository
	productQueryService ProductQueryService
	productCategoryRepo repository.ProductCategoryRepository
	coffeeBeanRepo      repository.CoffeeBeanRepository
	coffeeBrewRepo      repository.ProductCoffeeBrewRepository
	stockRepo           repository.StockRepository
}

func NewUpdateProductUseCase(i *do.Injector) *updateProductUseCase {
	return &updateProductUseCase{
		ctx:                 do.MustInvoke[context.Context](i),
		productRepo:         do.MustInvoke[repository.ProductRepository](i),
		productQueryService: do.MustInvoke[ProductQueryService](i),
		productCategoryRepo: do.MustInvoke[repository.ProductCategoryRepository](i),
		coffeeBeanRepo:      do.MustInvoke[repository.CoffeeBeanRepository](i),
		coffeeBrewRepo:      do.MustInvoke[repository.ProductCoffeeBrewRepository](i),
		stockRepo:           do.MustInvoke[repository.StockRepository](i),
	}
}

func (uc *updateProductUseCase) Execute(id string, param *ProductParam) error {
	ctx, cancel := context.WithTimeout(uc.ctx, CtxTimeoutDur)
	defer cancel()

	product, err := uc.productQueryService.FindById(ctx, id)
	if err != nil {
		return err
	}

	if len(param.ProductName) != 0 {
		if err := product.ProductName.Set(param.ProductName); err != nil {
			return errors.Join(err, ErrInvalidParam)
		}
	}
	if param.ProductCategoryId != product.ProductCategory.GetId().String() {
		productCategory, err := uc.productCategoryRepo.FindById(ctx, param.ProductCategoryId)
		if err != nil {
			return errors.Join(err, ErrInvalidParam)
		}
		product.ProductCategory = *productCategory
	}
	if model.ProductType(param.ProductType) != product.ProductType {
		product.ProductType = model.ProductType(param.ProductType)
	}

	// Only Coffee
	if param.CoffeeBeanId != product.CoffeeBean.GetId().String() {
		coffeeBean, err := uc.coffeeBeanRepo.FindById(ctx, param.CoffeeBeanId)
		if err != nil {
			return errors.Join(err, ErrInvalidParam)
		}
		product.CoffeeBean = coffeeBean
	}
	if model.ProductType(param.ProductType) == model.ProductType(model.Coffee) && param.CoffeeBrews != nil {
		// New
		newBrews := lo.Filter(param.CoffeeBrews, func(pBrew CoffeeBrewParam, _ int) bool {
			if len(pBrew.Id) == 0 {
				return true
			}
			return false
		})
		for _, pBrew := range newBrews {
			brew, err := model.NewProductCoffeeBrew(product.GetId(), pBrew.Name, pBrew.BeanQuantityGrams, pBrew.Amount)
			if err != nil {
				return errors.Join(err, ErrInvalidParam)
			}
			if err := uc.coffeeBrewRepo.Save(ctx, brew); err != nil {
				return err
			}
		}
		// Diff
		diffBrews := lo.Filter(param.CoffeeBrews, func(pBrew CoffeeBrewParam, _ int) bool {
			for _, brew := range *product.CoffeeBrews {
				if pBrew.Id == brew.GetId().String() && (pBrew.Name != brew.GetName() || pBrew.BeanQuantityGrams != brew.BeanQuantityGrams || pBrew.Amount != brew.Amount) {
					return true
				}
			}
			return false
		})
		for _, pBrew := range diffBrews {
			brew, err := uc.coffeeBrewRepo.FindById(ctx, pBrew.Id)
			if err != nil {
				return errors.Join(err, ErrInvalidParam)
			}
			if err := brew.SetName(pBrew.Name); err != nil {
				return errors.Join(err, ErrInvalidParam)
			}
			brew.BeanQuantityGrams = pBrew.BeanQuantityGrams
			brew.Amount = pBrew.Amount
			if err := uc.coffeeBrewRepo.Save(ctx, brew); err != nil {
				return err
			}
		}
		// Remove
		removedBrews := lo.Filter(*product.CoffeeBrews, func(brew *model.ProductCoffeeBrew, _ int) bool {
			for _, pBrew := range param.CoffeeBrews {
				if brew.GetId().String() == pBrew.Id {
					return false
				}
			}
			return true
		})
		for _, brew := range removedBrews {
			if err := uc.coffeeBrewRepo.Delete(ctx, brew.GetId().String()); err != nil {
				return err
			}
		}
	}

	// Only Other
	if param.Amount != 0 {
		product.SetAmount(param.Amount)
	}
	if model.ProductType(param.ProductType) == model.ProductType(model.Other) && param.StockId != product.Stock.GetId().String() {
		stock, err := uc.stockRepo.FindById(ctx, param.StockId)
		if err != nil {
			return errors.Join(err, ErrInvalidParam)
		}
		product.Stock = stock
	}

	if err := uc.productRepo.Save(ctx, product); err != nil {
		return err
	}

	return nil
}
