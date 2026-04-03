package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	Name         string
	Path         string
	HasArchCheck bool
}

// DiscoverServices finds all services under servicesDir that have mise.toml with arch:check task
func DiscoverServices(servicesDir, archTaskName string) ([]Service, error) {
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	var services []Service

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		servicePath := filepath.Join(servicesDir, entry.Name())
		miseTomlPath := filepath.Join(servicePath, "mise.toml")

		hasArchCheck, err := hasMiseArchCheck(miseTomlPath, archTaskName)
		if err != nil {
			return nil, fmt.Errorf("check arch:check task in %s: %w", miseTomlPath, err)
		}

		services = append(services, Service{
			Name:         entry.Name(),
			Path:         servicePath,
			HasArchCheck: hasArchCheck,
		})
	}

	return services, nil
}

// hasMiseArchCheck checks if mise.toml has arch:check task
func hasMiseArchCheck(miseTomlPath, archTaskName string) (bool, error) {
	file, err := os.Open(miseTomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("fail to open %s : %w", miseTomlPath, err)
	}
	defer file.Close()

	doubleQuoted := fmt.Sprintf(`"%s"`, archTaskName)
	singleQuoted := fmt.Sprintf("'%s'", archTaskName)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, doubleQuoted) || strings.Contains(line, singleQuoted) {
			return true, nil
		}
	}

	if err = scanner.Err(); err != nil {
		return false, fmt.Errorf("scanner error: %w", err)
	}

	return false, nil
}
