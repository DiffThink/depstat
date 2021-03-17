package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// analyzeDepsCmd represents the analyzeDeps command
var analyzeDepsCmd = &cobra.Command{
	Use:   "analyzeDeps",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("analyzeDeps called")

		// get flags
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return err
		}

		// get output of "go mod graph" in a string
		goModGraph := exec.Command("go", "mod", "graph")
		goModGraphOutput, err := goModGraph.Output()
		if err != nil {
			return err
		}
		goModGraphOutputString := string(goModGraphOutput)

		// create a graph of dependencies from that output
		depGraph := make(map[string][]string)
		scanner := bufio.NewScanner(strings.NewReader(goModGraphOutputString))

		// deps will store all the dependencies
		// since can't do slice.contains so better to use map
		deps := make(map[string]bool)
		mainModule := "notset"

		for scanner.Scan() {
			line := scanner.Text()
			words := strings.Fields(line)
			// remove versions
			words[0] = (strings.Split(words[0], "@"))[0]
			words[1] = (strings.Split(words[1], "@"))[0]
			if mainModule == "notset" {
				mainModule = words[0]
			}
			deps[words[0]] = true
			deps[words[1]] = true

			// we don't want to add the same dep again
			if !contains(depGraph[words[0]], words[1]) {
				depGraph[words[0]] = append(depGraph[words[0]], words[1])
			}
		}

		if verbose {
			fmt.Println("All dependencies:")
			for k := range deps {
				if k == mainModule {
					continue
				}
				fmt.Println(k)
			}
			fmt.Println()
		}

		// Prepare Dynamic Programming arrays for max depth
		// dp[k] = max depth of dependencies starting from dependency "k"
		dp := make(map[string]int)
		// visited array will make sure we don't have infinite recursion
		visited := make(map[string]bool)

		// values not in map will have their respective 0 value by default
		// so need to worry about terminal nodes
		for k := range deps {
			dp[k] = 0
			visited[k] = false
		}
		// longestPath[k] = u means that from dependency "k" going to
		// dependency "u" will result in the longest path
		longestPath := make(map[string]string)

		// maps are pass by reference in golang
		for k := range deps {
			if visited[k] == false {
				dfs(k, depGraph, dp, visited, longestPath)
			}
		}

		// for each dependency the DP array has the longest path starting
		// from that dependency

		// show the longest dependency chain (not working):
		// if verbose {
		// 	cur := mainModule
		// 	pathVisited := make(map[string]bool)
		// 	for dp[cur] != 0 {
		// 		pathVisited[cur] = true
		// 		fmt.Print(cur + " -> ")
		// 		nextDep := ""
		// 		for _, depOfCur := range depGraph[cur] {
		// 			if pathVisited[depOfCur] == false {
		// 				if dp[depOfCur] >= dp[nextDep] {
		// 					nextDep = depOfCur
		// 				}
		// 			}
		// 		}
		// 		cur = nextDep
		// 	}
		// 	fmt.Printf(cur)
		// 	fmt.Println()
		// }

		// also not working:
		if verbose {
			cur := mainModule
			// have visited array here too
			vis := make(map[string]bool)
			for vis[cur] == false {
				vis[cur] = true
				fmt.Print(cur + " -> ")
				cur = longestPath[cur]
				// if vis[cur] == true {
				// 	break
				// }
			}
		}

		// get values
		totalDeps := len(deps) - 1 // -1 for main module name
		maxDepth := dp[mainModule]
		directDeps := len(depGraph[mainModule])
		transitiveDeps := totalDeps - directDeps

		// create json
		outputObj := struct {
			TotalDeps int `json:"totalDependencies"`
			MaxDepth  int `json:"maxDepthOfDependencies"`
			TransDeps int `json:"transitiveDependencies"`
		}{
			TotalDeps: totalDeps,
			MaxDepth:  maxDepth,
			TransDeps: transitiveDeps,
		}
		outputRaw, err := json.Marshal(outputObj)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile("analysis.json", outputRaw, 0644)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyzeDepsCmd)
	analyzeDepsCmd.Flags().BoolP("verbose", "v", false, "Get additional details")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// analyzeDepsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// analyzeDepsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}