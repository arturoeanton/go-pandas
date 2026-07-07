// Command compat-report reads the compatibility matrices and prints
// summary numbers, so the coverage report never drifts from the matrices:
//
//	go run ./cmd/compat-report
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type tally struct {
	done, partial, planned, notSupported int
}

func (t tally) tracked() int { return t.done + t.partial + t.planned + t.notSupported }

func (t tally) implemented() int { return t.done + t.partial }

func (t tally) percent() float64 {
	if t.tracked() == 0 {
		return 0
	}
	return 100 * float64(t.implemented()) / float64(t.tracked())
}

// scanMatrix counts the Status column of every table row in a matrix file.
func scanMatrix(path string) (tally, error) {
	f, err := os.Open(path)
	if err != nil {
		return tally{}, err
	}
	defer f.Close()
	var t tally
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		// | api | go api | status | notes | -> 6 fragments
		if len(cells) < 5 {
			continue
		}
		switch strings.TrimSpace(cells[3]) {
		case "done":
			t.done++
		case "partial":
			t.partial++
		case "planned":
			t.planned++
		case "not_supported":
			t.notSupported++
		}
	}
	return t, scanner.Err()
}

func report(name, path string) error {
	t, err := scanMatrix(path)
	if err != nil {
		return err
	}
	fmt.Printf("%-8s tracked=%3d  done=%3d  partial=%3d  planned=%3d  not_supported=%2d  implemented=%.0f%%\n",
		name, t.tracked(), t.done, t.partial, t.planned, t.notSupported, t.percent())
	return nil
}

func main() {
	if err := report("pandas", "compat/pandas_matrix.md"); err != nil {
		fmt.Fprintln(os.Stderr, "pandas matrix:", err)
		os.Exit(1)
	}
	if err := report("numpy", "compat/numpy_matrix.md"); err != nil {
		fmt.Fprintln(os.Stderr, "numpy matrix:", err)
		os.Exit(1)
	}
	fmt.Println("\nStatuses count 'done' and 'partial' as implemented; percentages are")
	fmt.Println("relative to tracked rows. Full detail: compat/coverage_report.md")
}
