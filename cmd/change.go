package cmd

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/giantswarm/abu/money"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	MONTH_LOOKBACK  = 3
	MAX_CONCURRENCY = 5
	NUM_LINES       = 10
)

var changeCmd = &cobra.Command{
	Use:   "change",
	Short: "Print information on changes in costs",
	Run:   runChange,
}

func init() {
	rootCmd.AddCommand(changeCmd)
}

func runChange(cmd *cobra.Command, args []string) {
	now := aws.String(time.Now().AddDate(0, -MONTH_LOOKBACK, -time.Now().Day()+1).Format("2006-01-02"))
	lookback := aws.String(time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02"))

	type Request struct {
		AccountName *string
		AccountId   *string
		Service     *string
		Region      *string
		Start       *string
		End         *string
	}

	type Result struct {
		AccountName        *string
		AccountId          *string
		Service            *string
		Region             *string
		Start              *string
		End                *string
		CostAndUsageOutput *costexplorer.GetCostAndUsageOutput
	}

	requests := []Request{}

	listAccountsResult, err := organizationsSvc.ListAccounts(&organizations.ListAccountsInput{})
	if err != nil {
		log.Fatal(err)
	}

	describeRegionsResult, err := ec2Svc.DescribeRegions(nil)
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	regions := []*string{}
	for _, region := range describeRegionsResult.Regions {
		regions = append(regions, region.RegionName)
	}

	for _, account := range listAccountsResult.Accounts {
		getDimensionValuesResult, err := costExplorerSvc.GetDimensionValues(&costexplorer.GetDimensionValuesInput{
			Dimension: aws.String("SERVICE"),
			TimePeriod: &costexplorer.DateInterval{
				Start: now,
				End:   lookback,
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range getDimensionValuesResult.DimensionValues {
			service := v.Value

			for _, region := range regions {
				requests = append(requests, Request{
					AccountName: account.Name,
					AccountId:   account.Id,
					Service:     service,
					Region:      region,
					Start:       now,
					End:         lookback,
				})
			}
		}
	}

	var g errgroup.Group
	g.SetLimit(MAX_CONCURRENCY)

	resultsChannel := make(chan Result, MAX_CONCURRENCY+1)

	for _, request := range requests {
		r := request

		go g.Go(func() error {
			getCostAndUsageInput := &costexplorer.GetCostAndUsageInput{
				Filter: &costexplorer.Expression{
					And: []*costexplorer.Expression{
						{
							Dimensions: &costexplorer.DimensionValues{
								Key:    aws.String("LINKED_ACCOUNT"),
								Values: []*string{r.AccountId},
							},
						},
						{
							Dimensions: &costexplorer.DimensionValues{
								Key:    aws.String("SERVICE"),
								Values: []*string{r.Service},
							},
						},
						{
							Dimensions: &costexplorer.DimensionValues{
								Key:    aws.String("REGION"),
								Values: []*string{r.Region},
							},
						},
					},
				},
				Granularity: aws.String("MONTHLY"),
				Metrics:     []*string{aws.String("UnblendedCost")},
				TimePeriod: &costexplorer.DateInterval{
					Start: r.Start,
					End:   r.End,
				},
			}

			getCostAndUsageOutput, err := costExplorerSvc.GetCostAndUsage(getCostAndUsageInput)
			if err != nil {
				return err
			}

			resultsChannel <- Result{
				AccountName:        r.AccountName,
				AccountId:          r.AccountId,
				Service:            r.Service,
				Region:             r.Region,
				Start:              r.Start,
				End:                r.End,
				CostAndUsageOutput: getCostAndUsageOutput,
			}

			return nil
		})
	}

	go func() {
		g.Wait()
		if err := g.Wait(); err != nil {
			log.Fatal(err)
		}
		close(resultsChannel)
	}()

	results := []Result{}

	for result := range resultsChannel {
		results = append(results, result)
	}

	type Line struct {
		Name         string
		Id           string
		Service      string
		Region       string
		DollarCost   float64
		EuroCost     float64
		DollarChange float64
		EuroChange   float64
	}

	lines := []Line{}
	for _, result := range results {
		r := result.CostAndUsageOutput.ResultsByTime
		firstResult := r[0]
		lastResult := r[len(r)-1]

		firstDollarCost, err := money.CostExplorerResultByTimeToDollar(firstResult)
		if err != nil {
			log.Fatal(err)
		}
		firstEuroCost, err := money.CostExplorerResultByTimeToEuro(firstResult)
		if err != nil {
			log.Fatal(err)
		}

		lastDollarCost, err := money.CostExplorerResultByTimeToDollar(lastResult)
		if err != nil {
			log.Fatal(err)
		}
		lastEuroCost, err := money.CostExplorerResultByTimeToEuro(lastResult)
		if err != nil {
			log.Fatal(err)
		}

		dollarChange := lastDollarCost - firstDollarCost
		euroChange := lastEuroCost - firstEuroCost

		line := Line{
			Name:         *result.AccountName,
			Id:           *result.AccountId,
			Service:      *result.Service,
			Region:       *result.Region,
			DollarCost:   lastDollarCost,
			EuroCost:     lastEuroCost,
			DollarChange: dollarChange,
			EuroChange:   euroChange,
		}

		lines = append(lines, line)
	}

	slices.SortFunc(lines, func(a, b Line) int {
		return cmp.Compare(b.DollarChange, a.DollarChange)
	})

	if len(lines) > NUM_LINES {
		lines = lines[:NUM_LINES]
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		NAME_TITLE,
		ID_TITLE,
		SERVICE_TITLE,
		REGION_TITLE,
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
		DELTA_DOLLAR_TITLE,
		DELTA_ESTIMATED_EURO_TITLE,
	}, "\t"))

	for _, line := range lines {
		s := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t",
			line.Name,
			line.Id,
			line.Service,
			line.Region,
			money.Float64DollarToStringDollar(line.DollarCost),
			money.Float64EuroToStringEuro(line.EuroCost),
			money.Float64DollarToStringDollar(line.DollarChange),
			money.Float64EuroToStringEuro(line.EuroChange),
		)
		fmt.Fprintln(w, s)
	}

	w.Flush()
}
