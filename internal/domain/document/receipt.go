package document

import (
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Receipt struct {
	Id            bson.ObjectID `bson:"_id,omitempty"`
	ReceiptNumber string        `bson:"receipt_number"`
	InvoiceNumber string        `bson:"invoice_number"`
	Card          CardSummary   `bson:"card"`
	ProcessedAt   time.Time     `bson:"processed_at"`
}

type CardSummary struct {
	Brand string `bson:"brand"`
	Last4 string `bson:"last4"`
}

func NewReceiptFromProtMessage(dto *dto.PayWithCardResponse) (Receipt, error) {
	if err := validation.New().
		NotBlank("receiptNumber", dto.GetReceiptNumber()).
		NotBlank("invoiceNumber", dto.GetInvoiceNumber()).
		NotNil("cardSummary", dto.GetCard()).
		NotBlank("cardSummary.brand", dto.GetCard().GetBrand()).
		NotBlank("cardSummary.last4", dto.GetCard().GetLast4()).
		NotNil("processedAt", dto.GetProcessedAt()).Validate(); err != nil {

		return Receipt{}, err
	}

	return Receipt{
		Id:            bson.NewObjectID(),
		ReceiptNumber: dto.GetReceiptNumber(),
		InvoiceNumber: dto.GetInvoiceNumber(),
		Card: CardSummary{
			Brand: dto.GetCard().GetBrand(),
			Last4: dto.GetCard().GetLast4(),
		},
		ProcessedAt: dto.ProcessedAt.AsTime(),
	}, nil
}
