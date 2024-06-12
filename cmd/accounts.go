package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/spf13/cobra"

	"github.com/giantswarm/abu/money"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Print information on accounts",
	Run:   runAccounts,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
}

func runAccounts(cmd *cobra.Command, args []string) {
	result, err := organizationsSvc.ListAccounts(&organizations.ListAccountsInput{})
	if err != nil {
		log.Fatal(err)
	}

	type CostInfo struct {
		Id     string
		Dollar float64
		Euro   float64
	}

	type ForecastInfo struct {
		Id     string
		Dollar float64
		Euro   float64
	}

	costInfoChannel := make(chan CostInfo, len(result.Accounts))
	forecastInfoChannel := make(chan ForecastInfo, len(result.Accounts))

	var wg sync.WaitGroup

	for _, account := range result.Accounts {
		wg.Add(1)

		go func(account *organizations.Account, costInfoChannel chan CostInfo) {
			defer wg.Done()

			costInfo := CostInfo{
				Id: *account.Id,
			}

			getCostAndUsageInput := &costexplorer.GetCostAndUsageInput{
				Filter: &costexplorer.Expression{
					Dimensions: &costexplorer.DimensionValues{
						Key:    aws.String("LINKED_ACCOUNT"),
						Values: []*string{account.Id},
					},
				},
				Granularity: aws.String("MONTHLY"),
				GroupBy: []*costexplorer.GroupDefinition{
					{
						Type: aws.String("DIMENSION"),
						Key:  aws.String("LINKED_ACCOUNT"),
					},
				},
				Metrics: []*string{aws.String("UnblendedCost")},
				TimePeriod: &costexplorer.DateInterval{
					Start: aws.String(time.Now().AddDate(0, -1, -time.Now().Day()+1).Format("2006-01-02")),
					End:   aws.String(time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")),
				},
			}

			getCostAndUsageOutput, err := costExplorerSvc.GetCostAndUsage(getCostAndUsageInput)
			if err != nil {
				log.Fatal(err)
			}

			var group *costexplorer.Group

			for _, resultByTime := range getCostAndUsageOutput.ResultsByTime {
				for _, g := range resultByTime.Groups {
					accountId := *g.Keys[0]
					if accountId == *account.Id {
						group = g
					}
				}
			}

			dollar, err := money.CostExplorerGroupToDollar(group)
			if err != nil {
				log.Fatal(err)
			}

			euro, err := money.CostExplorerGroupToEuro(group)
			if err != nil {
				log.Fatal(err)
			}

			costInfo.Dollar = dollar
			costInfo.Euro = euro

			costInfoChannel <- costInfo
		}(account, costInfoChannel)

		wg.Add(1)

		go func(account *organizations.Account, ch chan ForecastInfo) {
			defer wg.Done()

			forecastInfo := ForecastInfo{
				Id: *account.Id,
			}

			getCostForecastInput := &costexplorer.GetCostForecastInput{
				Filter: &costexplorer.Expression{
					Dimensions: &costexplorer.DimensionValues{
						Key:    aws.String("LINKED_ACCOUNT"),
						Values: []*string{account.Id},
					},
				},
				Granularity: aws.String("MONTHLY"),
				Metric:      aws.String("UNBLENDED_COST"),
				TimePeriod: &costexplorer.DateInterval{
					Start: aws.String(time.Now().Format("2006-01-02")),
					End:   aws.String(time.Now().AddDate(0, 0, 1).Format("2006-01-02")),
				},
			}

			getCostForecastOutput, err := costExplorerSvc.GetCostForecast(getCostForecastInput)

			if err == nil {
				dollar, err := money.ForecastResultToDollar(getCostForecastOutput.ForecastResultsByTime[0])
				if err != nil {
					log.Fatal(err)
				}

				euro, err := money.ForecastResultToEuro(getCostForecastOutput.ForecastResultsByTime[0])
				if err != nil {
					log.Fatal(err)
				}

				forecastInfo.Dollar = dollar
				forecastInfo.Euro = euro
			}

			forecastInfoChannel <- forecastInfo
		}(account, forecastInfoChannel)
	}

	go func() {
		wg.Wait()

		close(costInfoChannel)
		close(forecastInfoChannel)
	}()

	costInfos := []CostInfo{}
	forecastInfos := []ForecastInfo{}

	for costInfo := range costInfoChannel {
		costInfos = append(costInfos, costInfo)
	}
	for forecastInfo := range forecastInfoChannel {
		forecastInfos = append(forecastInfos, forecastInfo)
	}

	type Line struct {
		Name      string
		Id        string
		Suspended string

		Dollar float64
		Euro   float64

		DollarForecast float64
		EuroForecast   float64

		DollarDelta float64
		EuroDelta   float64
	}

	lines := []Line{}
	for _, account := range result.Accounts {
		line := Line{
			Name:      *account.Name,
			Id:        *account.Id,
			Suspended: "NO",
		}

		if *account.Status == "SUSPENDED" {
			line.Suspended = "YES"
		}

		for _, costInfo := range costInfos {
			if costInfo.Id == line.Id {
				line.Dollar = costInfo.Dollar
				line.Euro = costInfo.Euro
			}
		}
		for _, forecastInfo := range forecastInfos {
			if forecastInfo.Id == line.Id {
				line.DollarForecast = forecastInfo.Dollar
				line.EuroForecast = forecastInfo.Euro
			}
		}

		line.DollarDelta = line.DollarForecast - line.Dollar
		line.EuroDelta = line.EuroForecast - line.Euro

		lines = append(lines, line)
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Name < lines[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		NAME_TITLE,
		ID_TITLE,
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
		FORECAST_DOLLAR_TITLE,
		FORECAST_ESTIMATED_EURO_TITLE,
		COST_FORECAST_DELTA_DOLLAR_TITLE,
		COST_FORECAST_DELTA_ESTIMATED_EURO_TITLE,
		SUSPENDED_TITLE,
	}, "\t"))

	for _, line := range lines {
		s := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			line.Name,
			line.Id,
			money.Float64DollarToStringDollar(line.Dollar),
			money.Float64EuroToStringEuro(line.Euro),
			money.Float64DollarToStringDollar(line.DollarForecast),
			money.Float64EuroToStringEuro(line.EuroForecast),
			money.Float64DollarToStringDollar(line.DollarDelta),
			money.Float64EuroToStringEuro(line.EuroDelta),
			line.Suspended,
		)
		fmt.Fprintln(w, s)
	}

	w.Flush()
}
