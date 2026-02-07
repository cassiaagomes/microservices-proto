package shipping_adapter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cassiaagomes/microservices-proto/golang/shipping"
	"github.com/cassiaagomes/microservices/order/internal/application/core/domain"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Adapter struct {
	client shipping.ShippingServiceClient
}

// Inicializa o adapter do Shipping
func NewAdapter(shippingServiceUrl string) (*Adapter, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(
			grpc_retry.WithCodes(codes.DeadlineExceeded, codes.ResourceExhausted, codes.Unavailable),
			grpc_retry.WithMax(5),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(time.Second)),
		)),
	}

	conn, err := grpc.Dial(shippingServiceUrl, opts...)
	if err != nil {
		return nil, err
	}

	client := shipping.NewShippingServiceClient(conn)
	return &Adapter{client: client}, nil
}

func (a *Adapter) Create(ctx context.Context, order *domain.Order) (int32, error) {
	var items []*shipping.ShippingItem

	for _, i := range order.OrderItems {
		// Proto espera int64, mas no domínio vem string
		pc, err := strconv.ParseInt(i.ProductCode, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("productCode inválido (%s): %w", i.ProductCode, err)
		}

		items = append(items, &shipping.ShippingItem{
			ProductCode: pc,
			Quantity:    int32(i.Quantity),
		})
	}

	resp, err := a.client.GetDeliveryEstimate(ctx, &shipping.ShippingRequest{
		OrderId: order.ID,
		Items:   items,
	})
	if err != nil {
		return 0, err
	}

	return resp.DeliveryDays, nil
}
