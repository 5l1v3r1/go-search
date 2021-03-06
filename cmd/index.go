package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/sundowndev/go-search/engine"
)

func init() {
	// Register command
	rootCmd.AddCommand(indexCmd)
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Add files to database indexation",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := engine.NewRedisClient(redisAddr, redisPort, "", 0)
		if err != nil {
			fmt.Println("Failed to connect to database", redisAddr, redisPort)
			os.Exit(1)
		}
		defer engine.Close(client)

		path := args[0]

		fmt.Printf("Walking %v...\n", path)

		files, err := engine.ScanDir(path)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			// Open File
			f, err := ioutil.ReadFile(file)
			if err != nil {
				panic(err)
			}

			if !engine.IsText(f) {
				continue
			}

			content := string(f)

			err = engine.AddFile(client, file, content)
			if err != nil {
				panic(err)
			}

			fmt.Println("Successfully indexed file", file)
		}
	},
}
