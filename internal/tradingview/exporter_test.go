package tradingview

import (
	"fmt"
	"github.com/jwtly10/tradebook/internal/account"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateTradePinescript(t *testing.T) {
	trades := []account.Trade{
		{
			ID:         1,
			Direction:  "LONG",
			EntryPrice: 23085.50,
			EntryTime:  time.Date(2025, 8, 4, 13, 45, 0, 0, time.UTC),
			ExitPrice:  23185.50,
			ExitTime:   time.Date(2025, 8, 4, 17, 0, 0, 0, time.UTC),
			PnL:        200.00,
			TakeProfit: 23185.50,
			StopLoss:   23085.50,
			ExitReason: "TAKE_PROFIT",
		},
	}

	pineCode := generateTradePinescript(trades)
	fmt.Println(pineCode)

	expected := `// ============================================
// TRADE VALIDATION MARKERS
// ============================================

t1_entry = time == timestamp("UTC", 2025, 8, 4, 13, 45)
plotshape(t1_entry, title="#1 LONG Entry", location=location.bottom, color=color.blue, style=shape.labelup, size=size.small, text="#1 LONG\nEntry: 23085.50000\nTP: 23185.50000\nSL: 23085.50000", textcolor=color.white)

t1_exit = time == timestamp("UTC", 2025, 8, 4, 17, 0)
plotshape(t1_exit, title="#1 EXIT", location=location.top, color=color.green, style=shape.labeldown, size=size.small, text="#1 EXIT\nExit: 23185.50000\nTAKE_PROFIT", textcolor=color.white)

`

	// This was an idea I had. Keeping so I don't forget about it
	//	expected := `// ============================================
	//// TRADE VALIDATION MARKERS
	//// ============================================
	//
	//// Trade 1 Entry
	//t1_entry = time == timestamp("UTC", 2025, 8, 4, 13, 45)
	//plot(t1_entry ? 1 : na, "#1 LONG Entry: 23085.50", color.blue, style=plot.style_circles, linewidth=3)
	//plotshape(t1_entry, location=location.belowbar, color=color.blue, style=shape.triangleup, size=size.small)
	//
	//// Trade 1 Exit
	//t1_exit = time == timestamp("UTC", 2025, 8, 4, 17, 0)
	//plot(t1_exit ? 1 : na, "#1 EXIT: 23185.50 | P&L: £200.00 | TAKE_PROFIT", color.green, style=plot.style_circles, linewidth=3)
	//plotshape(t1_exit, location=location.abovebar, color=color.green, style=shape.triangledown, size=size.small)
	//
	//`
	//

	assert.Equal(t, expected, pineCode)
}
