/*

 Program  written by Matt Tesauro <matt.tesauro@owasp.org>
 as part of the OWASP WTE project

 This file, jerry-curl, is a wrapper for the curl command allowing
 default arguements to be easly added when invoking curl.  This is
 partularly useful when headers need to be added to every request
 such as HTTP basic auth or talking to REST APIs. 

 jerry-curl is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 jerry-curl is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with jerry-curl.  If not, see <http://www.gnu.org/licenses/>.

 Sat, 03 Nov 2012 17:25:25 -0600
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Global declarations
var configDir string = ".jerry-curl"
var configFile string = "jerry-curl.config"
var home string

func main() {
	// Check if curl is installed
	curl := curlCheck()

	// Check if the configuration file exits and create if necessary
	createConfig()

	// Report argument collisions with curl
	argClash(os.Args)

	// Parse the command line arguments
	jerryArgs, curlArgs := parseArgs(os.Args)

	// Iterate through jerryArgs to see if alternate config was set
	newConfig := ""
	for _, value := range jerryArgs {
		if (strings.Contains(value, "--config")) || (strings.Contains(value, "-c")) {
			sp := strings.Index(value, " ")
			newConfig = value[sp+1:]
		}
	}

	// Read config file
	base, extraArgs := readConfig(newConfig)

	// Interate through the maps and finish generating the final command
	curlCmd, showOnly := genCurlCmd(jerryArgs, curlArgs, base)

	// Generate curl command with arguments
	finalCmd := append(extraArgs, curlCmd...)

	// Show command based on command line arguments and exit
	if showOnly {
		fmt.Println("Here is the curl command which would run:")
		fmt.Printf("%s %s\n", curl, strings.Join(finalCmd, " "))
		os.Exit(0)
	}

	fmt.Printf("final command is %v\n\n", finalCmd)

	// Build the resulting curl command to run
	//cmd := exec.Command(curl, finalCmd...)
	cmd := exec.Command("curl", finalCmd...)

	fmt.Printf("final command is %v\n\n", finalCmd)

	// Catch stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:\n\t%v", err)
		os.Exit(0)
	}
	// Catch stderr
	errorReader, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error:\n\t%v", err)
		os.Exit(0)
	}
	// Start executing the curl command
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
		os.Exit(0)
	}

	// Use IO copy to print curl's stdout
	io.Copy(os.Stdout, stdout)
	// Buffer stderr for possible future display
	buf := new(bytes.Buffer)
	buf.ReadFrom(errorReader)

	fmt.Printf("Args in cmd is %v\n", cmd.Args)

	// Wait for curl to complete
	cmd.Wait()

	fmt.Printf("curl exit = %s|\n\n", cmd.ProcessState.String())

	// Check for a curl error
	if cmd.ProcessState.String() != "exit status 0" {
		fmt.Println("A curl error occured:\n")
		fmt.Printf("\t%s\n", buf.String())
	}

}

func curlCheck() string {
	path, err := exec.LookPath("curl")
	if err != nil {
		fmt.Println("The curl command must be installed and in your path.")
		fmt.Println("Please install curl to enjoy all of jerry-curl")
		os.Exit(0)
	}

	return path
}

func createConfig() {
	// Setup a few things
	home = os.Getenv("HOME")
	if home == "" {
		fmt.Println("The environmental varaible HOME needs to be set to your home directory")
		//Todo - allow a command-line to set home?
		os.Exit(0)
	}

	// TODO - simplify this to a single path.join with all three strings as args
	config := path.Join(home, configDir)
	configFile := path.Join(config, configFile)

	// Check if the config file exists and return early if its present
	file, _ := os.Stat(configFile)
	if file != nil {
		return
	}

	// Create the config directory if it doesn't exist
	_, err := os.Stat(config)
	if err != nil {
		// Need to create config directory
		err = os.Mkdir(config, 0700)
		if err != nil {
			fmt.Printf("Unable to create config directory at %s\n", config)
		}
	}

	// Create a default config file
	defaultConfig := `# Some example of items to put in this config file
# 
# For repeated requests to the same URL:
# BASE=http://www.example.com
#    NOTE:  BASE is the only config option done as a key=value pair.
#           All others are simply one command line option per line.
#           Make sure the line starts with "BASE="
# 
# Proxy curl commands:
# --proxy 127.0.0.1:8080 
# 
# Allow insecure SSL:
# --insecure
#
# Include headers in the output
# -include
# 
# Set an Auth header
# -H "X-Auth-Token: 55555555-5555-5555-5555-555555555555"
#
# Set accepts header 
# -H "Accept: application/json"
# 
# Set content-type header
# -H "Content-Type: application/json"
# `

	// Write a default config
	fileBytes := []byte(defaultConfig)
	err = ioutil.WriteFile(configFile, fileBytes, 0600)
	if err != nil {
		fmt.Println("Unable to write config file")
		fmt.Printf("Please check permissions of %s\n", config)
		os.Exit(0)
	}

}

func argClash(args []string) {
	// Run through the map, counting matches on command line arguements
	// used by both jerry-curl and curl or two of the same jerry-curl arguments
	countConfig := 0
	countC := 0
	countShow := 0
	countS := 0
	countUrl := 0
	countU := 0
	message := "Error(s):"

	for _, arg := range args {
		switch arg {
		case "--config":
			countConfig++
		case "-c":
			countC++
		case "--show":
			countShow++
		case "-s":
			countS++
		case "--url-path":
			countUrl++
		case "-u":
			countU++
		}
	}

	// Add any necessary error messages for argument clashes between jerry-curl and curl
	if countConfig >= 2 {
		message += "\n\t* Option --config used twice and is ambiguous."
		message += "\n\t  If you want the curl version, use -K instead"
	}
	if countC >= 2 {
		message += "\n\t* Option -c used twice and is ambiguous."
		message += "\n\t  If you want the curl version, use --cookie-jar <file name> instead"
	}
	if countS >= 2 {
		message += "\n\t* Option -s used twice and is ambiguous."
		message += "\n\t  If you want the curl version, use --silent instead"
	}
	if countU >= 2 {
		message += "\n\t* Option -u used twice and is ambiguous."
		message += "\n\t  If you want the curl version, use --user <user:password> instead"
	}
	// Check for using two forms of the same argument
	if (countConfig >= 1) && (countC >= 1) {
		message += "\n\t* Both option --config and -c used and is ambiguous."
		message += "\n\t  jerry-curl accepts either --config OR -c but not both"
	}
	if (countShow >= 1) && (countS >= 1) {
		message += "\n\t* Both option --show and -s used and is ambiguous."
		message += "\n\t  jerry-curl accepts either --show OR -s but not both"
	}
	if (countUrl >= 1) && (countU >= 1) {
		message += "\n\t* Both option --url-path and -u used and is ambiguous."
		message += "\n\t  jerry-curl accepts either --url-path OR -u but not both"
	}

	// Print appropriate message(s) and exit
	// if ((countConfig + countC + countS + countU) > 0)
	if len(message) > 9 {
		fmt.Printf("%s\n", message)
		os.Exit(0)
	}
}

func parseArgs(args []string) (map[int]string, []string) {
	// Setup maps to hold the arguments for both jerry-curl and curl
	jerryArgs := make(map[int]string)
	var curlArgs []string
	// skip set to true to skip adding the os.Args[0] from being processed
	skip := true
	var next string

	// iterate through args sent and place jerry-curl args into one map
	// all other args are passed through to curl via curlArgs
	for n, arg := range args {
		missing := false
		switch arg {
		case "-c", "--config":
			// Check for an option without an arguement at the end of the command line
			// which would make args[n+1:n+2] out of bounds
			if len(args) < n+2 {
				missing = true
			} else {
				next = strings.Join(args[n+1:n+2], "")
			}

			if missing || next[0] == 45 {
				fmt.Println("Error:")
				fmt.Printf(" jerry-curl's %s option requires an agrument\n", arg)
				fmt.Printf(" such as %s ./path/to/config\n\n", arg)
				os.Exit(0)
			}

			jerryArgs[n] = arg + " " + next
			// skip adding the next arg to curlArgs
			skip = true
		case "-s", "--show":
			jerryArgs[n] = arg
		case "-u", "--url-path":
			// As above, check args length before slicing
			if len(args) < n+2 {
				missing = true
			} else {
				next = strings.Join(args[n+1:n+2], "")
			}

			if missing || next[0] == 45 {
				fmt.Println("Error:")
				fmt.Printf(" jerry-curl's %s option requires an agrument\n", arg)
				fmt.Printf(" such as %s /path/to/add/to/url\n\n", arg)
				os.Exit(0)
			}
			jerryArgs[n] = arg + " " + next
			// skip adding the next arg to curlArgs
			skip = true
		case "-h", "--help":
			//If we're here, print the help and exit
			printHelp()
		default:
			if skip == false {
				curlArgs = append(curlArgs, arg)
			} else {
				skip = false
			}
		}
	}

	return jerryArgs, curlArgs

}

func printHelp() {
	// Print jerry-curls help message and exit
	fmt.Println("Usage: jerry-curl [-h|--help]")
	fmt.Println("   or: jerry-curl [jerry-curl options] [optional arguments for curl]")
	fmt.Println("")
	fmt.Println(" jerry-curl is a wrapper for the curl command which adds ")
	fmt.Println(" options from a configuration file and the command line")
	fmt.Println(" allowing for short repeated curl calls.")
	fmt.Println("")
	fmt.Println(" jerry-curl works by calling curl like the below:")
	fmt.Println("  curl [config options] [BASE][URLPATH] [command-line arguments]")
	fmt.Println("")
	fmt.Println(" jerry-curl commandline options:")
	fmt.Println("   -c, --config FILE         Select a different config file from the default")
	fmt.Println("                             which is $HOME/.jerry-curl/jerry-curl.config")
	fmt.Println("                               Example: jerry-curl --config=./my-custom-config")
	fmt.Println("   -s, --show                Show the curl command - DO NOT EXECUTE IT")
	fmt.Println("   -u, --url-path URLPATH    Set a path to append to the base URL")
	fmt.Println("                               Example: jerry-curl --url-path=/app/path/here")
	fmt.Println("   -h, --help                help, aka this message")
	fmt.Println("")
	fmt.Println("    Note: options --config, -c, -s -u are used by both jerry-curl and curl.  If")
	fmt.Println("    You want those sent to curl, please use their alternate forms.  Using --show")
	fmt.Println("    can help diagose if jerry-curl or curl is recieving a command-line option")
	fmt.Println("")
	fmt.Println(" If no config file exists, one will be created with commented examples in directory named ")
	fmt.Println(" .jerry-curl in your home directory when jerry-curl is run for the first time.")
	fmt.Println("")
	os.Exit(0)
}

func readConfig(newConfig string) (string, []string) {
	var extras []string // extra args from config file
	var base string     // base URL from config file
	var config string = ""

	// Check to see if the config option was used
	if len(newConfig) > 0 {
		config = newConfig
	} else {
		// Setup a few things to use the default config location
		home = os.Getenv("HOME")
		if home == "" {
			fmt.Println("The environmental varaible HOME needs to be set to your home directory")
			//Todo - allow a command-line to set home?
			os.Exit(0)
		}
		config = path.Join(home, configDir, configFile)
	}

	// Read configuration file to pull out any configured items
	file, err := os.Open(config)
	if err != nil {
		fmt.Printf("Error reading configuration file at %s\n", config)
		fmt.Printf("\t OS error: %s\n", err.Error())
		os.Exit(0)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	for err == nil {
		// Handle lines that are not comments
		if strings.Index(line, "#") != 0 {
			n := strings.Index(line, "BASE=")
			// Set base or else its an extra arg for curl
			if n == 0 {
				base = strings.TrimRight(line[n+5:], "\n")
			} else {
				extras = append(extras, strings.TrimRight(line, "\n"))
			}
		}

		line, err = reader.ReadString('\n')
	}
	if err != io.EOF {
		fmt.Println(err)
		os.Exit(0)
	}

	return base, extras
}

func genCurlCmd(jerry map[int]string, curl []string, base string) ([]string, bool) {
	// Some setup
	show := false
	path := ""
	var sCurl []string

	// Iterate through jerry pulling out any necessary bits
	for _, arg := range jerry {

		if (strings.Index(arg, "--show") == 0) || (strings.Index(arg, "-s") == 0) {
			show = true
		}
		if (strings.Index(arg, "--url-path") == 0) || (strings.Index(arg, "-u")) == 0 {
			path = strings.TrimLeft(arg, "--url-path ")
			path = strings.TrimLeft(arg, "-u ")
		}
	}

	// Iterate through curl arguements and sanity check them
	// for quoted arguments
	for _, item := range curl[:] {
		if strings.Index(item, "\"") >= 0 {
			newItem := "'" + item + "'"
			sCurl = append(sCurl, newItem)
		} else {
			sCurl = append(sCurl, item)
		}

	}

	url := base + path

	fmt.Printf("URL in method =%s|\n\n", url)

	return append(sCurl, url), show
}
