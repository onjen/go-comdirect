package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jsattler/go-comdirect/pkg/comdirect"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	transactionHeader = []string{"DESCRIPTION", "BOOKING DATE", "STATUS", "TYPE", "VALUE", "UNIT"}
	transactionCmd    = &cobra.Command{
		Use:   "transaction",
		Short: "list account transactions",
		Args:  cobra.MinimumNArgs(1),
		Run:   transaction,
	}
)

func transaction(cmd *cobra.Command, args []string) {
	client := initClient()
	ctx, cancel := contextWithTimeout()
	defer cancel()

	options := comdirect.EmptyOptions()
	options.Add(comdirect.PagingCountQueryKey, countFlag)
	options.Add(comdirect.PagingFirstQueryKey, indexFlag)
	transactions, err := client.Transactions(ctx, args[0], options)
	if err != nil {
		log.Fatal(err)
	}

	switch formatFlag {
	case "json":
		printJSON(transactions)
	case "markdown":
		printTransactionTable(transactions)
	case "csv":
		printTransactionCSV(transactions)
	default:
		printTransactionTable(transactions)
	}
}

func printJSON(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func cleanRemittanceInfo(remittanceInfo string) string {
	var cleanedInfo string
	// every line is 2 (line number, zero padded) + 35 (text) characters long
	for i := 0; i < len(remittanceInfo)/37; i++ {
		currentLine := remittanceInfo[(i*37)+2 : ((i + 1) * 37)]
		if strings.Contains(currentLine, "End-to-End-Ref") {
			break
		}
		cleanedInfo += strings.TrimRight(currentLine, " ")
		cleanedInfo += " "
	}
	return cleanedInfo
}

func printTransactionCSV(transactions *comdirect.AccountTransactions) {
	table := csv.NewWriter(os.Stdout)
	table.Write(transactionHeader)
	for _, t := range transactions.Values {
		holderName := t.Remitter.HolderName
		var description string
		if len(holderName) > 0 {
			description += holderName + " "
		}
		if len(t.Creditor.HolderName) > 0 {
			description += t.Creditor.HolderName + " "
		}
		description += cleanRemittanceInfo(t.RemittanceInfo)
		if t.BookingStatus == "BOOKED" {
			table.Write([]string{description, t.BookingDate, t.BookingStatus, t.TransactionType.Text, formatAmountValue(t.Amount), t.Amount.Unit})
		}
	}
	table.Flush()
}
func printTransactionTable(transactions *comdirect.AccountTransactions) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(transactionHeader)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetCaption(true, fmt.Sprintf("%d out of %d", len(transactions.Values), transactions.Paging.Matches))
	for _, t := range transactions.Values {
		holderName := t.Remitter.HolderName
		var description string
		if len(holderName) > 0 {
			description += holderName + " "
		}
		if len(t.Creditor.HolderName) > 0 {
			description += t.Creditor.HolderName + " "
		}
		description += cleanRemittanceInfo(t.RemittanceInfo)
		if len(description) > 30 {
			description = description[:30]
		}
		if t.BookingStatus == "BOOKED" {
			table.Append([]string{description, t.BookingDate, t.BookingStatus, t.TransactionType.Text, formatAmountValue(t.Amount), t.Amount.Unit})
		}
	}
	table.Render()
}
