package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/difftool"
	"github.com/grafana/plugin-validator/pkg/sourcemap"
	"github.com/grafana/plugin-validator/pkg/utils"
)

var options = map[string]*string{
	"archiveUri":     flag.String("archiveUri", "", "The URI of the plugin archive. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a zip file."),
	"sourceCodeUri":  flag.String("sourceCodeUri", "", "The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."),
	"nonInteractive": flag.String("nonInteractive", "n", "If set to 'y', the command will not ask for user input and will use the default values for all the options."),
}

func main() {
	flag.Parse()

	// resolve the pluginPath from options
	pluginPath, archiveCleanup, err := getLocalPathFromArchiveOption(*options["archiveUri"])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer archiveCleanup()

	// resolve the sourceCodePath from options
	sourceCodePath, sourceCodeCleanup, err := getLocalPathFromSourceCodeOption(*options["sourceCodeUri"])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer sourceCodeCleanup()

	// extract the source map into a temporal folder
	sourceCodeMapPath := filepath.Join(pluginPath, "module.js.map")

	// get the plugin id from the archive
	pluginID, err := utils.GetPluginId(pluginPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// compare the source map with the source code
	report, err := difftool.CompareSourceMapToSourceCode(pluginID, sourceCodeMapPath, filepath.Join(sourceCodePath, "src"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(report.GeneratePrintableReport())
	if report.TotalDifferences > 0 && *options["nonInteractive"] == "n" {
		promptToSeeDiff(pluginID, sourceCodeMapPath, sourceCodePath)
	}

}

func promptToSeeDiff(pluginID, sourceCodeMapPath string, sourceCodePath string) {

	var answer string
	fmt.Println("Do you want to see the differences in your diff tool?")
	fmt.Println("Be aware the original source code will contain more files than the source map such as images, readme files and typescript typefiles.")
	fmt.Println("\n\nIt is recommended to install meld (https://meldmerge.org/) to see the differences.")
	fmt.Print("Open diff tool? (y/n): ")
	fmt.Scanln(&answer)

	if answer == "y" || answer == "Y" {

		systemDiffTool, err := getSystemDiffTool()
		if err != nil {
			color.Red(err.Error())
			suggestDiffTool()
			return
		}

		sourceCodeMapPath, err := sourcemap.ExtractSourceMapToPath(pluginID, sourceCodeMapPath)
		if err != nil {
			fmt.Println("Error extracting source map to file system")
			fmt.Println(err)
			os.Exit(1)
		}

		command := fmt.Sprintf(systemDiffTool, sourceCodeMapPath, filepath.Join(sourceCodePath, "src"))

		// run the command
		fmt.Println("Running command: ", command)
		cmd := exec.Command("sh", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println("Error running command")
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func getSystemDiffTool() (string, error) {
	var err error

	// try meld
	_, err = exec.LookPath("meld")
	if err == nil {
		return "meld %s %s", nil
	}

	// try colordiff
	_, err = exec.LookPath("colordiff")
	if err == nil {
		return "colordiff -bur %s %s", nil
	}

	// try old good diff
	_, err = exec.LookPath("diff")
	if err == nil {
		return "diff -bur %s %s", nil
	}

	return "", fmt.Errorf("could not find a diff tool. Please install one")
}

func suggestDiffTool() {
	color.Yellow("\nPlease install a diff tool such as meld.")
	// detect if in a mac
	if runtime.GOOS == "darwin" {
		fmt.Println("You can install meld with: brew install meld")
	} else if runtime.GOOS == "linux" {
		fmt.Println("You can install meld using your package manager. e.g.:")
		fmt.Println("In Ubuntu you can install it with:      sudo apt install meld")
		fmt.Println("In Fedora you can install it with:      sudo dnf install meld")
		fmt.Println("In Arch Linux you can install it with:  sudo pacman -S meld")
	}
}
