package cmd

import (
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/spf13/cobra"
)

var (
	NAME_TITLE      = "NAME"
	ID_TITLE        = "ID"
	MONTH_TITLE     = "MONTH"
	SUSPENDED_TITLE = "SUSP."
	SERVICE_TITLE   = "SERVICE"
	REGION_TITLE    = "REGION"

	DOLLAR         = "($)"
	ESTIMATED_EURO = "(~€)"

	BILL                      = "BILL"
	BILL_DOLLAR_TITLE         = strings.Join([]string{BILL, DOLLAR}, " ")
	BILL_ESTIMATED_EURO_TITLE = strings.Join([]string{BILL, ESTIMATED_EURO}, " ")

	COST                      = "COST"
	COST_DOLLAR_TITLE         = strings.Join([]string{COST, DOLLAR}, " ")
	COST_ESTIMATED_EURO_TITLE = strings.Join([]string{COST, ESTIMATED_EURO}, " ")

	FORECAST                      = "FORECAST"
	FORECAST_DOLLAR_TITLE         = strings.Join([]string{FORECAST, DOLLAR}, " ")
	FORECAST_ESTIMATED_EURO_TITLE = strings.Join([]string{FORECAST, ESTIMATED_EURO}, " ")

	BUDGET                      = "BUDGET"
	BUDGET_DOLLAR_TITLE         = strings.Join([]string{BUDGET, DOLLAR}, " ")
	BUDGET_ESTIMATED_EURO_TITLE = strings.Join([]string{BUDGET, ESTIMATED_EURO}, " ")

	DELTA                      = "Δ"
	DELTA_DOLLAR_TITLE         = strings.Join([]string{DELTA, DOLLAR}, " ")
	DELTA_ESTIMATED_EURO_TITLE = strings.Join([]string{DELTA, ESTIMATED_EURO}, " ")

	BUDGET_FORECAST_DELTA_DOLLAR_TITLE       = strings.Join([]string{BUDGET, "/", FORECAST, DELTA, DOLLAR}, " ")
	BUDGET_FORECAST_DELTA_EURO_TITLE         = strings.Join([]string{BUDGET, "/", FORECAST, DELTA, ESTIMATED_EURO}, " ")
	BILL_FORECAST_DELTA_DOLLAR_TITLE         = strings.Join([]string{COST, "/", FORECAST, DELTA, DOLLAR}, " ")
	BILL_FORECAST_DELTA_ESTIMATED_EURO_TITLE = strings.Join([]string{COST, "/", FORECAST, DELTA, ESTIMATED_EURO}, " ")
)

var (
	sess *session.Session

	budgetSvc        *budgets.Budgets
	ec2Svc           *ec2.EC2
	costExplorerSvc  *costexplorer.CostExplorer
	organizationsSvc *organizations.Organizations
)

var rootCmd = &cobra.Command{
	Use:   "abu",
	Short: "abu is a utility for AWS billing",
}

func init() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String("eu-west-1")},
		SharedConfigState: session.SharedConfigEnable,
	}))

	budgetSvc = budgets.New(sess)
	ec2Svc = ec2.New(sess)
	costExplorerSvc = costexplorer.New(sess)
	organizationsSvc = organizations.New(sess)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func AWSMonthToString(m string) (string, error) {
	mp, err := time.Parse("2006-01-02", m)
	if err != nil {
		log.Fatal(err)
	}

	return mp.Month().String(), nil
}
