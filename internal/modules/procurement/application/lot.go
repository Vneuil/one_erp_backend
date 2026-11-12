package application

import (
	"fmt"
	"time"
)

// GenerateLotNumber builds the lot number a received material carries from the GRN on:
// LOT-<YYMMDD>-<receipt sequence>-<line>. It is stable, sortable and unique per receipt line.
func GenerateLotNumber(t time.Time, receiptSeq, line int) string {
	return fmt.Sprintf("LOT-%s-%04d-%02d", t.Format("060102"), receiptSeq, line)
}
