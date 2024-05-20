package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/spf13/cobra"

	"github.com/giantswarm/abu/money"
)

var billsCmd = &cobra.Command{
	Use:   "bills",
	Short: "Print information on bills",
	Run:   runBills,
}

func init() {
	rootCmd.AddCommand(billsCmd)
}

func runBills(cmd *cobra.Command, args []string) {
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(time.Now().AddDate(0, -6, -time.Now().Day()+1).Format("2006-01-02")),
			End:   aws.String(time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")),
		},
		Granularity: aws.String("MONTHLY"),
		Metrics:     []*string{aws.String("UnblendedCost")},
	}

	result, err := costExplorerSvc.GetCostAndUsage(input)
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		MONTH_TITLE,
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
	}, "\t"))

	for i := len(result.ResultsByTime) - 1; i >= 0; i-- {
		resultByTime := result.ResultsByTime[i]

		if *resultByTime.Estimated {
			continue
		}

		month, err := AWSMonthToString(*resultByTime.TimePeriod.Start)
		if err != nil {
			log.Fatal(err)
		}

		dollar, err := money.CostExplorerResultByTimeToDollar(resultByTime)
		if err != nil {
			log.Fatal(err)
		}

		euro, err := money.CostExplorerResultByTimeToEuro(resultByTime)
		if err != nil {
			log.Fatal(err)
		}

		s := fmt.Sprintf(
			"%s\t%s\t%s",
			month,
			money.Float64DollarToStringDollar(dollar),
			money.Float64EuroToStringEuro(euro),
		)
		fmt.Fprintln(w, s)
	}

	w.Flush()
}
