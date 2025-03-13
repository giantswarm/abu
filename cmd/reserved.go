package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/spf13/cobra"

	"github.com/giantswarm/abu/money"
)

var reservedCmd = &cobra.Command{
	Use:   "reserved",
	Short: "Print information on Reserved Instances",
	Run:   runReserved,
}

func init() {
	rootCmd.AddCommand(reservedCmd)
}

func runReserved(cmd *cobra.Command, args []string) {
	// Get the start and end dates for the previous month
	now := time.Now()
	endDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	startDate := time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, endDate.Location())

	// Format dates for AWS API
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.AddDate(0, 0, 1).Format("2006-01-02")

	fmt.Printf("Reserved Instances Report for %s %d\n\n", startDate.Month().String(), startDate.Year())

	// Get reservation utilization by region
	fmt.Println("Reservation Utilization by Region:")
	err := getRegionUtilization(startDateStr, endDateStr)
	if err != nil {
		fmt.Printf("Error retrieving region utilization: %v\n", err)
		fmt.Println("Make sure you have valid AWS credentials configured.")
		return
	}

	// Get reservation utilization by instance type
	fmt.Println("\nReservation Utilization by Instance Type:")
	err = getInstanceTypeUtilization(startDateStr, endDateStr)
	if err != nil {
		fmt.Printf("Error retrieving instance type utilization: %v\n", err)
		fmt.Println("Make sure you have valid AWS credentials configured.")
	}
}

func getRegionUtilization(startDate, endDate string) error {
	input := &costexplorer.GetReservationUtilizationInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(startDate),
			End:   aws.String(endDate),
		},
		GroupBy: []*costexplorer.GroupDefinition{
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("REGION"),
			},
		},
	}

	result, err := costExplorerSvc.GetReservationUtilization(input)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		REGION_TITLE,
		"PURCHASED HOURS",
		"UTILIZED HOURS",
		"UTILIZATION %",
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
	}, "\t"))

	for _, utilization := range result.UtilizationsByTime {
		for _, group := range utilization.Groups {
			region := *group.Attributes["region"]
			purchasedHours, _ := strconv.ParseFloat(*group.Utilization.PurchasedHours, 64)
			usedHours, _ := strconv.ParseFloat(*group.Utilization.TotalActualHours, 64)
			utilizationPct, _ := strconv.ParseFloat(*group.Utilization.UtilizationPercentage, 64)
			
			amortizedCost, _ := strconv.ParseFloat(*group.Utilization.TotalAmortizedFee, 64)
			amortizedCostEuro := money.DollarToEuro(amortizedCost)

			fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f%%\t%s\t%s\n",
				region,
				purchasedHours,
				usedHours,
				utilizationPct,
				money.Float64DollarToStringDollar(amortizedCost),
				money.Float64EuroToStringEuro(amortizedCostEuro),
			)
		}
	}

	w.Flush()
	return nil
}

func getInstanceTypeUtilization(startDate, endDate string) error {
	input := &costexplorer.GetReservationUtilizationInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(startDate),
			End:   aws.String(endDate),
		},
		GroupBy: []*costexplorer.GroupDefinition{
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("INSTANCE_TYPE"),
			},
		},
	}

	result, err := costExplorerSvc.GetReservationUtilization(input)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		"INSTANCE TYPE",
		"PURCHASED HOURS",
		"UTILIZED HOURS",
		"UTILIZATION %",
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
	}, "\t"))

	for _, utilization := range result.UtilizationsByTime {
		for _, group := range utilization.Groups {
			instanceType := *group.Attributes["instance_type"]
			purchasedHours, _ := strconv.ParseFloat(*group.Utilization.PurchasedHours, 64)
			usedHours, _ := strconv.ParseFloat(*group.Utilization.TotalActualHours, 64)
			utilizationPct, _ := strconv.ParseFloat(*group.Utilization.UtilizationPercentage, 64)
			
			amortizedCost, _ := strconv.ParseFloat(*group.Utilization.TotalAmortizedFee, 64)
			amortizedCostEuro := money.DollarToEuro(amortizedCost)

			fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f%%\t%s\t%s\n",
				instanceType,
				purchasedHours,
				usedHours,
				utilizationPct,
				money.Float64DollarToStringDollar(amortizedCost),
				money.Float64EuroToStringEuro(amortizedCostEuro),
			)
		}
	}

	w.Flush()
	return nil
}
