package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/spf13/cobra"

	"github.com/giantswarm/abu/money"
)

var budgetCmd = &cobra.Command{
	Use:   "budgets",
	Short: "Print information on budgets",
	Run:   runBudget,
}

func init() {
	rootCmd.AddCommand(budgetCmd)
}

func runBudget(cmd *cobra.Command, args []string) {
	organizationResult, err := organizationsSvc.DescribeOrganization(&organizations.DescribeOrganizationInput{})
	if err != nil {
		log.Fatal(err)
	}

	budgetResult, err := budgetSvc.DescribeBudgets(&budgets.DescribeBudgetsInput{
		AccountId: aws.String(*organizationResult.Organization.MasterAccountId),
	})
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		NAME_TITLE,
		BUDGET_DOLLAR_TITLE,
		BUDGET_ESTIMATED_EURO_TITLE,
		COST_DOLLAR_TITLE,
		COST_ESTIMATED_EURO_TITLE,
		FORECAST_DOLLAR_TITLE,
		FORECAST_ESTIMATED_EURO_TITLE,
		FORECAST_DELTA_DOLLAR_TITLE,
		FORECAST_DELTA_EURO_TITLE,
	}, "\t"))

	for _, budget := range budgetResult.Budgets {
		limitDollar, err := money.BudgetLimitToDollar(budget)
		if err != nil {
			log.Fatal(err)
		}

		limitEuro, err := money.BudgetLimitToEuro(budget)
		if err != nil {
			log.Fatal(err)
		}

		spendDollar, err := money.BudgetSpendToDollar(budget)
		if err != nil {
			log.Fatal(err)
		}

		spendEuro, err := money.BudgetSpendToEuro(budget)
		if err != nil {
			log.Fatal(err)
		}

		forecastDollar, err := money.BudgetForecastToDollar(budget)
		if err != nil {
			log.Fatal(err)
		}

		forecastEuro, err := money.BudgetForecastToEuro(budget)
		if err != nil {
			log.Fatal(err)
		}

		forecastDeltaDollar := limitDollar - forecastDollar
		forecastDeltaEuro := limitEuro - forecastEuro

		s := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			*budget.BudgetName,
			money.Float64DollarToStringDollar(limitDollar),
			money.Float64EuroToStringEuro(limitEuro),
			money.Float64DollarToStringDollar(spendDollar),
			money.Float64EuroToStringEuro(spendEuro),
			money.Float64DollarToStringDollar(forecastDollar),
			money.Float64EuroToStringEuro(forecastEuro),
			money.Float64DollarToStringDollar(forecastDeltaDollar),
			money.Float64EuroToStringEuro(forecastDeltaEuro),
		)
		fmt.Fprintln(w, s)
	}

	w.Flush()
}
