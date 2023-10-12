package orderlink_server

import (
	"connectrpc.com/connect"
	"context"
	"github.com/KaguraGateway/cafelogos-grpc/pkg/orderlink"
	"github.com/KaguraGateway/cafelogos-grpc/pkg/orderlink/orderlinkconnect"
	"github.com/KaguraGateway/cafelogos-pos-backend/domain/model"
	"github.com/KaguraGateway/cafelogos-pos-backend/domain/repository"
	"github.com/samber/do"
	"github.com/samber/lo"
)

type orderHookOrderLink struct {
	client orderlinkconnect.OrderLinkServiceClient
}

func NewOrderHookOrderLink(i *do.Injector) (repository.OrderHookRepository, error) {
	return &orderHookOrderLink{client: do.MustInvoke[orderlinkconnect.OrderLinkServiceClient](i)}, nil
}

func (i *orderHookOrderLink) PostOrder(ctx context.Context, order *model.Order, ticket *model.OrderTicket) error {
	_, err := i.client.PostOrder(ctx, connect.NewRequest(&orderlink.PostOrderInput{
		OrderId: order.GetId(),
		OrderAt: order.GetOrderAt().Format("2006-01-02T15:04:05Z"),
		Items: lo.Map(order.GetOrderItems(), func(item model.OrderItem, _ int) *orderlink.PostOrderItemInput {
			coffeeBrew := item.GetCoffeeHowToBrew()
			var coffeeBrewId = ""
			if &coffeeBrew != nil {
				coffeeBrewId = coffeeBrew.GetId()
			}
			return &orderlink.PostOrderItemInput{
				ProductId:    item.GetProductId(),
				CoffeeBrewId: coffeeBrewId,
				Quantity:     uint32(item.Quantity),
			}
		}),
		Type:       orderlink.PostOrderInput_OrderType(order.GetOrderType()),
		TicketId:   ticket.GetTicketId(),
		TicketAddr: ticket.GetTicketAddr(),
	}))
	if err != nil {
		return err
	}
	return nil
}
