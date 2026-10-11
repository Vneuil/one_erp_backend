package application

import (
	"encoding/xml"
	"fmt"
	"math"

	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/divinecoid/one-backend/internal/shared/taxid"
)

// The XML below follows the Coretax "TaxInvoiceBulk" import layout as far as it
// is publicly documented (DJP publishes no XSD): element names and order are
// those of the converter's sample file. It has NOT been imported into a live
// Coretax; compare with the XML the DJP converter produces for one faktur.

// coretaxMaxInvoicesPerFile is the number of Faktur Pajak Coretax accepts in one XML file.
const coretaxMaxInvoicesPerFile = 1000

type xmlGoodService struct {
	Opt           string `xml:"Opt"`
	Code          string `xml:"Code"`
	Name          string `xml:"Name"`
	Unit          string `xml:"Unit"`
	Price         string `xml:"Price"`
	Qty           string `xml:"Qty"`
	TotalDiscount string `xml:"TotalDiscount"`
	TaxBase       string `xml:"TaxBase"`
	OtherTaxBase  string `xml:"OtherTaxBase"`
	VATRate       string `xml:"VATRate"`
	VAT           string `xml:"VAT"`
	STLGRate      string `xml:"STLGRate"`
	STLG          string `xml:"STLG"`
}

type xmlTaxInvoice struct {
	TaxInvoiceDate      string           `xml:"TaxInvoiceDate"`
	TaxInvoiceOpt       string           `xml:"TaxInvoiceOpt"`
	TrxCode             string           `xml:"TrxCode"`
	AddInfo             string           `xml:"AddInfo"`
	CustomDoc           string           `xml:"CustomDoc"`
	CustomDocMonthYear  string           `xml:"CustomDocMonthYear"`
	RefDesc             string           `xml:"RefDesc"`
	FacilityStamp       string           `xml:"FacilityStamp"`
	SellerIDTKU         string           `xml:"SellerIDTKU"`
	BuyerTin            string           `xml:"BuyerTin"`
	BuyerDocument       string           `xml:"BuyerDocument"`
	BuyerCountry        string           `xml:"BuyerCountry"`
	BuyerDocumentNumber string           `xml:"BuyerDocumentNumber"`
	BuyerName           string           `xml:"BuyerName"`
	BuyerAdress         string           `xml:"BuyerAdress"` // sic: the template's spelling
	BuyerEmail          string           `xml:"BuyerEmail"`
	BuyerIDTKU          string           `xml:"BuyerIDTKU"`
	Goods               []xmlGoodService `xml:"ListOfGoodService>GoodService"`
}

type xmlBulk struct {
	XMLName xml.Name        `xml:"TaxInvoiceBulk"`
	XSI     string          `xml:"xmlns:xsi,attr"`
	Schema  string          `xml:"xsi:noNamespaceSchemaLocation,attr"`
	TIN     string          `xml:"TIN"`
	Invoice []xmlTaxInvoice `xml:"ListOfTaxInvoice>TaxInvoice"`
}

// ceilCent rounds up to a whole sen, so Price x Qty never falls below the DPP
// and the difference can be carried as a (non-negative) TotalDiscount.
func ceilCent(v float64) float64 { return math.Ceil(v*100-1e-9) / 100 }

// CoretaxXML builds the import file for issued Faktur Pajak Keluaran.
func CoretaxXML(sellerNPWP string, invoices []domain.TaxInvoice) ([]byte, error) {
	seller := taxid.NPWP16(sellerNPWP)
	doc := xmlBulk{XSI: "http://www.w3.org/2001/XMLSchema-instance", Schema: "TaxInvoice.xsd", TIN: seller}
	for _, t := range invoices {
		opt := "Normal"
		if t.Revision > 0 {
			opt = "Pengganti"
		}
		inv := xmlTaxInvoice{
			TaxInvoiceDate: t.Date, TaxInvoiceOpt: opt, TrxCode: t.TransactionCode, AddInfo: t.Notes, RefDesc: t.SourceReference,
			SellerIDTKU: seller + "000000", BuyerCountry: "IDN", BuyerName: t.CounterpartyName, BuyerAdress: t.CounterpartyAddress, BuyerEmail: t.CounterpartyEmail,
		}
		if t.CounterpartyNPWP != "" {
			buyer := taxid.NPWP16(t.CounterpartyNPWP)
			inv.BuyerTin, inv.BuyerDocument, inv.BuyerIDTKU = buyer, "TIN", buyer+"000000"
		} else {
			inv.BuyerTin, inv.BuyerDocument, inv.BuyerDocumentNumber, inv.BuyerIDTKU = "0000000000000000", "National ID", t.CounterpartyNIK, "000000"
		}
		for _, l := range t.Lines {
			price := ceilCent(l.TaxBase / nonZero(l.Quantity))
			discount := round2(price*l.Quantity - l.TaxBase)
			if discount < 0 {
				discount = 0
			}
			other := 0.0
			if t.TransactionCode == "04" || l.DPPOtherValue > 0 {
				other = l.DPPOtherValue
			}
			inv.Goods = append(inv.Goods, xmlGoodService{
				Opt: "A", Code: coretaxNoGoodsCode, Name: l.Description, Unit: coretaxUnitPiece,
				Price: money(price), Qty: money(l.Quantity), TotalDiscount: money(discount), TaxBase: money(l.TaxBase),
				OtherTaxBase: money(other), VATRate: fmt.Sprintf("%g", t.VATRate), VAT: money(l.VATAmount), STLGRate: "0", STLG: "0.00",
			})
		}
		doc.Invoice = append(doc.Invoice, inv)
	}
	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func nonZero(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
