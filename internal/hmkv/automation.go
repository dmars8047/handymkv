package hmkv

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type AutomationParamSource string

const (
	ParamSourcePrompt     AutomationParamSource = "prompt"
	ParamSourceStatic     AutomationParamSource = "static"
	ParamSourceHMKVOutput AutomationParamSource = "hmkv_output"
)

var validParamSources = []string{
	string(ParamSourcePrompt),
	string(ParamSourceStatic),
	string(ParamSourceHMKVOutput),
}

var validHMKVKeys = []string{
	"hb_output_dir",
	"mkv_output_dir",
	"run_duration",
	"raw_files_deleted",
	"title_count",
	"total_raw_size",
	"total_encoded_size",
}

type AutomationParam struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Source       AutomationParamSource `json:"source"`
	DefaultValue string                `json:"default_value,omitempty"`
	HMKVKey      string                `json:"hmkv_key,omitempty"`
}

type Automation struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Params  []AutomationParam `json:"params,omitempty"`
}

// validateAutomationName rejects names that could escape the automations directory.
func validateAutomationName(name string) error {
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("automation name '%s' contains invalid characters (no path separators or '..' allowed)", name)
	}
	return nil
}

// getAutomationsDir returns the OS-appropriate automations directory path.
func getAutomationsDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
		return filepath.Join(appData, "handymkv", "automations"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "handymkv", "automations"), nil
}

// ListAutomations reads all automation JSON files from the automations directory.
func ListAutomations() ([]Automation, error) {
	dir, err := getAutomationsDir()
	if err != nil {
		return nil, err
	}

	pattern := filepath.Join(dir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error scanning automations directory: %w", err)
	}

	var automations []Automation
	for _, f := range files {
		a, err := readAutomationFile(f)
		if err != nil {
			fmt.Printf("Warning: could not load automation file '%s': %v\n", filepath.Base(f), err)
			continue
		}
		automations = append(automations, *a)
	}

	return automations, nil
}

// LoadAutomation reads a single automation by name.
func LoadAutomation(name string) (*Automation, error) {
	if err := validateAutomationName(name); err != nil {
		return nil, err
	}

	dir, err := getAutomationsDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, name+".json")
	return readAutomationFile(filePath)
}

func readAutomationFile(path string) (*Automation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a Automation
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	if a.Command == "" {
		return nil, fmt.Errorf("automation '%s' has an empty command field", a.Name)
	}
	return &a, nil
}

// SaveAutomation writes an automation JSON file to the automations directory.
func SaveAutomation(a *Automation) error {
	dir, err := getAutomationsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0740); err != nil {
		return fmt.Errorf("could not create automations directory: %w", err)
	}

	filePath := filepath.Join(dir, a.Name+".json")

	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal automation: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0640); err != nil {
		return fmt.Errorf("could not write automation file: %w", err)
	}

	return nil
}

// DeleteAutomation deletes an automation file after confirmation.
func DeleteAutomation(name string) error {
	if err := validateAutomationName(name); err != nil {
		return err
	}

	dir, err := getAutomationsDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, name+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Automation '%s' not found.\n\n", name)
		return nil
	}

	fmt.Printf("Delete automation '%s'? [y/N]: ", name)

	if strings.ToLower(readLine()) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("could not delete automation: %w", err)
	}

	fmt.Printf("Automation '%s' deleted.\n\n", name)
	return nil
}

// PrintAutomations lists all saved automations.
func PrintAutomations() error {
	automations, err := ListAutomations()
	if err != nil {
		return err
	}

	if len(automations) == 0 {
		fmt.Println("No automations found.")
		fmt.Println("Run 'handymkv automations create' to create one.")
		fmt.Println()
		return nil
	}

	fmt.Printf("Saved Automations:\n\n")

	for i, a := range automations {
		paramCount := len(a.Params)
		paramLabel := "params"
		if paramCount == 1 {
			paramLabel = "param"
		}
		fmt.Printf("  %d. %s - %s (%d %s)\n", i+1, a.Name, a.Command, paramCount, paramLabel)
	}

	fmt.Printf("\nRun 'handymkv automations show <name>' to view details.\n\n")
	return nil
}

// PrintAutomation shows the details of a single automation.
func PrintAutomation(name string) error {
	a, err := LoadAutomation(name)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Automation '%s' not found.\n\n", name)
			return nil
		}
		return err
	}

	fmt.Printf("Automation: %s\n", a.Name)
	fmt.Printf("Command:    %s\n", a.Command)

	if len(a.Params) == 0 {
		fmt.Printf("Parameters: none\n")
	} else {
		fmt.Printf("Parameters:\n\n")
		for _, p := range a.Params {
			envVarName := "HMKV_PARAM_" + strings.ToUpper(p.Name)
			fmt.Printf("  %s -> %s\n", p.Name, envVarName)
			fmt.Printf("    Description: %s\n", p.Description)
			fmt.Printf("    Source:      %s\n", p.Source)

			switch p.Source {
			case ParamSourcePrompt:
				if p.DefaultValue != "" {
					fmt.Printf("    Default:     %s\n", p.DefaultValue)
				}
			case ParamSourceStatic:
				fmt.Printf("    Value:       %s\n", p.DefaultValue)
			case ParamSourceHMKVOutput:
				fmt.Printf("    HMKV Key:    %s\n", p.HMKVKey)
			}

			fmt.Println()
		}
	}

	fmt.Println()
	return nil
}

// CreateAutomation runs the interactive wizard to create a new automation.
func CreateAutomation() error {
	fmt.Println("Create a new automation")
	fmt.Println()
	fmt.Println("Note: Automation scripts receive parameters as environment variables.")
	fmt.Println("Each parameter is available as HMKV_PARAM_<UPPERCASED_NAME>.")
	fmt.Println()

	name := promptForString(
		"Automation name:",
		"A short identifier for this automation.",
		"",
		nil,
	)

	if name == "" {
		fmt.Println("Name is required.")
		return nil
	}

	// Sanitize name for use as filename
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")

	if err := validateAutomationName(name); err != nil {
		fmt.Printf("Invalid name: %v\n", err)
		return nil
	}

	command := promptForString(
		"Command to execute:",
		"The script or command that will be run after encoding completes.",
		"",
		nil,
	)

	if command == "" {
		fmt.Println("Command is required.")
		return nil
	}

	var params []AutomationParam

	for {
		addParam := promptForBool("Add a parameter?", "", false)

		if !addParam {
			break
		}

		clear()

		param := AutomationParam{}

		param.Name = promptForString("Parameter name:", "A short identifier (becomes HMKV_PARAM_<UPPERCASED_NAME>).", "", nil)
		if param.Name == "" {
			fmt.Println("Parameter name is required. Skipping.")
			continue
		}

		param.Description = promptForString("Description:", "Explain what this parameter is for.", "", nil)

		source := promptForSelection("Parameter source:", validParamSources)
		param.Source = AutomationParamSource(source)

		switch param.Source {
		case ParamSourcePrompt:
			param.DefaultValue = promptForString("Default value (optional):", "Shown to user when prompted. Leave blank for no default.", "", nil)
		case ParamSourceStatic:
			param.DefaultValue = promptForString("Static value:", "The hardcoded value for this parameter.", "", nil)
			if param.DefaultValue == "" {
				fmt.Println("Default value is required. Skipping parameter.")
				continue
			}
		case ParamSourceHMKVOutput:
			param.HMKVKey = promptForSelection("HandyMKV output key:", validHMKVKeys)
		}

		params = append(params, param)
		clear()
	}

	a := &Automation{
		Name:    name,
		Command: command,
		Params:  params,
	}

	if err := SaveAutomation(a); err != nil {
		return fmt.Errorf("could not save automation: %w", err)
	}

	dir, _ := getAutomationsDir()
	fmt.Printf("Automation '%s' saved to %s\n\n", a.Name, filepath.Join(dir, a.Name+".json"))
	return nil
}

// resolvePreRunParams collects prompt/env/default param values before the run starts.
// Deduplicates by param name across automations.
func resolvePreRunParams(automations []Automation) (map[string]string, error) {
	resolved := make(map[string]string)

	for _, a := range automations {
		for _, p := range a.Params {
			if _, exists := resolved[p.Name]; exists {
				continue
			}

			switch p.Source {
			case ParamSourcePrompt:
				val := promptForString(
					fmt.Sprintf("[%s] %s:", a.Name, p.Description),
					"",
					p.DefaultValue,
					nil,
				)
				resolved[p.Name] = val

			case ParamSourceStatic:
				resolved[p.Name] = p.DefaultValue

			case ParamSourceHMKVOutput:
				// Resolved after the run completes; skip here
				continue
			}
		}
	}

	return resolved, nil
}

// buildRunOutputData maps hmkv_key values to run output data.
func buildRunOutputData(
	hbDir string,
	mkvDir string,
	duration time.Duration,
	titleCount int,
	rawDeleted bool,
	totalRaw int64,
	totalEncoded int64,
) map[string]string {
	rawDeletedStr := "false"
	if rawDeleted {
		rawDeletedStr = "true"
	}

	return map[string]string{
		"hb_output_dir":      hbDir,
		"mkv_output_dir":     mkvDir,
		"run_duration":       formatTimeElapsedString(duration),
		"raw_files_deleted":  rawDeletedStr,
		"title_count":        fmt.Sprintf("%d", titleCount),
		"total_raw_size":     fmt.Sprintf("%d", totalRaw),
		"total_encoded_size": fmt.Sprintf("%d", totalEncoded),
	}
}

// RunAutomations executes selected automations with resolved parameters.
// Returns manifest entries recording each automation's name, command, and resolved param values.
func RunAutomations(automations []Automation, preRunParams map[string]string, outputData map[string]string) []manifestAutomation {
	if len(automations) == 0 {
		return nil
	}

	fmt.Println("\nRunning automations...")

	var entries []manifestAutomation

	for _, a := range automations {
		// Build environment and collect resolved params for manifest
		env := os.Environ()
		var manifestParams []manifestAutomationParam

		for _, p := range a.Params {
			envKey := "HMKV_PARAM_" + strings.ToUpper(p.Name)
			var val string

			if p.Source == ParamSourceHMKVOutput {
				val = outputData[p.HMKVKey]
			} else {
				val = preRunParams[p.Name]
			}

			env = append(env, fmt.Sprintf("%s=%s", envKey, val))
			manifestParams = append(manifestParams, manifestAutomationParam{
				Name:  p.Name,
				Value: val,
			})
		}

		// Execute via shell
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", a.Command)
		} else {
			cmd = exec.Command("sh", "-c", a.Command)
		}

		cmd.Env = env

		exitCode := 0
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
				fmt.Printf("  %s: FAILED (exit code %d)\n", a.Name, exitCode)
			} else {
				exitCode = -1
				fmt.Printf("  %s: FAILED (%v)\n", a.Name, err)
			}
		} else {
			fmt.Printf("  %s: OK\n", a.Name)
		}

		entries = append(entries, manifestAutomation{
			Name:     a.Name,
			Command:  a.Command,
			Params:   manifestParams,
			ExitCode: exitCode,
		})
	}

	fmt.Println()
	return entries
}
