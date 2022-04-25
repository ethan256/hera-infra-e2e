package assist

import (
	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/assister"
)

var (
	assistActual   string
	assistExpected string
	metricURL      string
)

func init() {
	Assist.Flags().StringVarP(&assistActual, "assist-actual", "", "", "the actual data file name")
	Assist.Flags().StringVarP(&assistExpected, "assist-expected", "", "", "the expected file name")
	Assist.Flags().StringVarP(&metricURL, "url", "", "", "metric query url")
}

var Assist = &cobra.Command{
	Use:   "assist",
	Short: "assist generate expected data by actual data",
	RunE: func(cmd *cobra.Command, args []string) error {
		return assister.ActualDataToExpected(assistActual, assistExpected, metricURL)
	},
}
