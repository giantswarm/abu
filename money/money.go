package money

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/leekchan/accounting"
)

var (
	dollarToEuro = 0.919606 // As of 2024-05-20, to use as fallback
)

func init() {
	attemptUpdateDollarToEuro()
}

func attemptUpdateDollarToEuro() {
	type response struct {
		Rates map[string]float64 `json:"rates"`
	}

	httpClient := http.Client{Timeout: time.Second * 1}

	r, err := httpClient.Get("https://open.er-api.com/v6/latest/USD")
	if err != nil {
		return
	}
	defer r.Body.Close()

	var target response
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		return
	}

	val, ok := target.Rates["EUR"]
	if !ok {
		return
	}
	dollarToEuro = val
}

func DollarToEuro(d float64) float64 {
	return d * dollarToEuro
}

func CostExplorerResultByTimeToDollar(resultByTime *costexplorer.ResultByTime) (float64, error) {
	if resultByTime == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*resultByTime.Total["UnblendedCost"].Amount, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func CostExplorerResultByTimeToEuro(resultByTime *costexplorer.ResultByTime) (float64, error) {
	dollar, err := CostExplorerResultByTimeToDollar(resultByTime)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func CostExplorerGroupToDollar(group *costexplorer.Group) (float64, error) {
	if group == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*group.Metrics["UnblendedCost"].Amount, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func CostExplorerGroupToEuro(group *costexplorer.Group) (float64, error) {
	dollar, err := CostExplorerGroupToDollar(group)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func ForecastResultToDollar(forecastResult *costexplorer.ForecastResult) (float64, error) {
	if forecastResult == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*forecastResult.MeanValue, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func ForecastResultToEuro(forecastResult *costexplorer.ForecastResult) (float64, error) {
	dollar, err := ForecastResultToDollar(forecastResult)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func BudgetSpendToDollar(budget *budgets.Budget) (float64, error) {
	if budget == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*budget.CalculatedSpend.ActualSpend.Amount, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func BudgetSpendToEuro(budget *budgets.Budget) (float64, error) {
	dollar, err := BudgetSpendToDollar(budget)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func BudgetForecastToDollar(budget *budgets.Budget) (float64, error) {
	if budget == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*budget.CalculatedSpend.ForecastedSpend.Amount, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func BudgetForecastToEuro(budget *budgets.Budget) (float64, error) {
	dollar, err := BudgetForecastToDollar(budget)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func BudgetLimitToDollar(budget *budgets.Budget) (float64, error) {
	if budget == nil {
		return 0, nil
	}

	dollar, err := strconv.ParseFloat(*budget.BudgetLimit.Amount, 64)
	if err != nil {
		return 0, err
	}

	return dollar, nil
}

func BudgetLimitToEuro(budget *budgets.Budget) (float64, error) {
	dollar, err := BudgetLimitToDollar(budget)
	if err != nil {
		return 0, err
	}

	euro := DollarToEuro(dollar)

	return euro, nil
}

func TruncateString(s string) string {
	return fmt.Sprintf("%10s", s)
}

func Float64DollarToStringDollar(f float64) string {
	ac := accounting.Accounting{Symbol: "$", Precision: 2, Thousand: ",", Decimal: "."}

	s := ac.FormatMoney(f)
	s = TruncateString(s)

	return s
}

func Float64EuroToStringEuro(f float64) string {
	ac := accounting.Accounting{Symbol: "€", Precision: 2, Thousand: ",", Decimal: "."}

	s := ac.FormatMoney(f)
	s = TruncateString(s)

	return s
}
