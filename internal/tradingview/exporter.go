package tradingview

import (
	"fmt"
	"github.com/jwtly10/tradebook/internal/account"
	"log/slog"
	"os"
	"strings"
	"time"
)

func allowDump() bool {
	// Get OS Env for dump DEBUG_DUMP=1 etc
	debugDump := os.Getenv("DEBUG_DUMP")
	if debugDump == "1" {
		slog.Info("DEBUG_DUMP=1, dumping to stderr")
		return true
	}

	return false
}

func DumpPineScript(trades []account.Trade) {
	if !allowDump() {
		return
	}

	pineCode := generateTradePinescript(trades)
	fmt.Println(pineCode)
}

// GenerateTradePinescript generates the Pine Script code for visualizing trades on a chart.
// It accepts a slice of Trade objects and returns a string containing Pine Script with markers
// for entries and exits, coded with relevant details like timestamps, prices, PnL, and reasons.
func generateTradePinescript(trades []account.Trade) string {
	var sb strings.Builder

	sb.WriteString("// ============================================\n")
	sb.WriteString("// TRADE VALIDATION MARKERS\n")
	sb.WriteString("// ============================================\n\n")

	for _, trade := range trades {
		// Entry marker
		entryTimestamp := formatPineTimestamp(trade.EntryTime)
		entryText := fmt.Sprintf("#%d %s\\nEntry: %.5f\\nTP: %.5f\\nSL: %.5f",
			trade.ID, trade.Direction, trade.EntryPrice, trade.TakeProfit, trade.StopLoss)

		sb.WriteString(fmt.Sprintf("t%d_entry = time == %s\n", trade.ID, entryTimestamp))
		sb.WriteString(fmt.Sprintf("plotshape(t%d_entry, title=\"#%d %s Entry\", location=location.bottom, color=color.blue, style=shape.labelup, size=size.small, text=\"%s\", textcolor=color.white)\n\n",
			trade.ID, trade.ID, trade.Direction, entryText))

		// Exit marker
		exitTimestamp := formatPineTimestamp(trade.ExitTime)
		exitColor := "color.green"
		if trade.ExitReason == "STOP_LOSS" {
			exitColor = "color.red"
		}
		exitText := fmt.Sprintf("#%d EXIT\\nExit: %.5f\\n%s",
			trade.ID, trade.ExitPrice, trade.ExitReason)

		sb.WriteString(fmt.Sprintf("t%d_exit = time == %s\n", trade.ID, exitTimestamp))
		sb.WriteString(fmt.Sprintf("plotshape(t%d_exit, title=\"#%d EXIT\", location=location.top, color=%s, style=shape.labeldown, size=size.small, text=\"%s\", textcolor=color.white)\n\n",
			trade.ID, trade.ID, exitColor, exitText))
	}

	return sb.String()
}

func formatPineTimestamp(t time.Time) string {
	utc := t.UTC()
	return fmt.Sprintf("timestamp(\"UTC\", %d, %d, %d, %d, %d)",
		utc.Year(), int(utc.Month()), utc.Day(), utc.Hour(), utc.Minute())
}
