package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts",
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	result, err := organizationsSvc.ListAccounts(&organizations.ListAccountsInput{})
	if err != nil {
		log.Fatal(err)
	}

	accounts := result.Accounts
	sort.Slice(accounts, func(i, j int) bool {
		return *accounts[i].Name < *accounts[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintln(w, strings.Join([]string{
		NAME_TITLE,
		ID_TITLE,
		SUSPENDED_TITLE,
	}, "\t"))

	for _, account := range accounts {
		suspended := "NO"
		if *account.Status == "SUSPENDED" {
			suspended = "YES"
		}

		s := fmt.Sprintf("%s\t%s\t%s", *account.Name, *account.Id, suspended)

		fmt.Fprintln(w, s)
	}

	w.Flush()
}
