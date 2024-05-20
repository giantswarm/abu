package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Print the URL to switch accounts",
	Run:   runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: abu switch <account-name|account-id>")
		return
	}

	roleName := os.Getenv("ABU_SWITCH_ROLE_NAME")
	if roleName == "" {
		fmt.Println("Please set the environment variable ABU_SWITCH_ROLE_NAME to the name of the role to use for switching roles")
		return
	}

	result, err := organizationsSvc.ListAccounts(&organizations.ListAccountsInput{})
	if err != nil {
		log.Fatal(err)
	}

	for _, account := range result.Accounts {
		if *account.Name == args[0] || *account.Id == args[0] {
			if *account.Status == "SUSPENDED" {
				fmt.Println("Account is suspended")
				return
			}

			url := fmt.Sprintf(
				"https://signin.aws.amazon.com/switchrole?account=%s&roleName=%s&displayName=%s",
				*account.Id,
				roleName,
				fmt.Sprintf("%s-%s", *account.Name, *account.Id),
			)

			fmt.Println(url)
		}
	}
}
