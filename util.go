package nib

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gofrs/uuid"
)

// CommandError carries the exit code a failure deserves, so a command can
// report misuse or a missing reference without knowing how the program exits.
type CommandError struct {
	Code    int
	Message string
}

func (err CommandError) Error() string {
	return err.Message
}

// UsageError describes a command that could not be understood: an unknown
// UsageError describes a command that could not be understood, as distinct from
// one that was understood and failed.
func UsageError(format string, a ...any) error {
	return CommandError{Code: EXIT_USAGE, Message: fmt.Sprintf(format, a...)}
}

// NoMatchError describes a reference -- typically an ID -- that matched nothing.
func NoMatchError(format string, a ...any) error {
	return CommandError{Code: EXIT_NO_MATCH, Message: fmt.Sprintf(format, a...)}
}

// NoOpError describes a request that was understood but changed nothing.
func NoOpError(format string, a ...any) error {
	return CommandError{Code: EXIT_NO_OP, Message: fmt.Sprintf(format, a...)}
}

// ExitError prints err and exits with the code it carries, or 1 if it carries
// none.
func ExitError(err error) {
	var commandErr CommandError
	if errors.As(err, &commandErr) {
		exitWith(commandErr.Code, "%s", err)
	}

	exitWith(EXIT_FAILURE, "%s", err)
}

func ExitFail(format string, a ...any) {
	exitWith(EXIT_FAILURE, format, a...)
}

// ExitUsage exits on a command line that could not be understood.
func ExitUsage(format string, a ...any) {
	exitWith(EXIT_USAGE, format, a...)
}

func exitWith(code int, format string, a ...any) {
	fmt.Fprintln(os.Stderr, sgr("31")+fmt.Sprintf(format, a...)+sgr("0"))
	os.Exit(code)
}

func ConfirmOrAbort(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+" [y/n] ", a...)

	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		// No answer available, so nothing was agreed to. Treating end of input
		// as a decline is what keeps an unattended run from proceeding.
		exitWith(EXIT_NO_OP, "aborted")
	}

	// Normalize input: remove CR/LF/whitespace and compare in lowercase
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "y" || normalized == "yes" {
		return
	}

	// declining leaves everything as it was
	exitWith(EXIT_NO_OP, "aborted")
}

func MustGetUUID4String() string {
	// does not match docs...
	u, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return u.String()
}

func IsValidUUID4String(str string) bool {
	_, err := uuid.FromString(str)

	return err == nil
}

func IsValidPriority(priority string) bool {
	return map[string]bool{
		PRIORITY_CRITICAL: true,
		PRIORITY_HIGH:     true,
		PRIORITY_NORMAL:   true,
		PRIORITY_LOW:      true,
	}[priority]
}

func IsValidStatus(status string) bool {
	return StrSliceContains(ALL_STATUSES, status)
}

func ParseDueDateArg(dueStr string) (dateFilter string, dueDate time.Time) {
	parts := strings.SplitN(dueStr, ":", 2)
	if len(parts) != 2 {
		ExitUsage("invalid due query format: " + dueStr + "\n" +
			"expected format: due:YYYY-MM-DD, due:MM-DD, due:DD, due:next-monday, due:today, etc.")
	}
	if parts[1] == "overdue" {
		dateFilter = "before"
		dueDate = startOfDay(time.Now())
		return dateFilter, dueDate
	}
	tagParts := strings.SplitN(parts[0], ".", 2)
	if len(tagParts) == 2 {
		dateFilter = tagParts[1]

		dateFilters := map[string]struct{}{"after": {}, "before": {}, "on": {}, "in": {}}
		_, ok := dateFilters[dateFilter]
		if !ok && dateFilter != "" {
			ExitUsage("invalid date filter format: " + dateFilter + "\n" +
				"valid filters are: after, before, on, in")
		}

	} else {
		dateFilter = ""
	}
	dueDate = ParseStrToDate(parts[1])
	return dateFilter, dueDate
}

func SumInts(vals ...int) int {
	var total int

	for _, v := range vals {
		total += v
	}

	return total
}

func RunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// MakeTempFilename encodes the task ID and a truncated portion of a task
// summary into a string suitable for passing to ioutil.TempFile.
func MakeTempFilename(id int, summary, ext string) string {
	truncated := make([]rune, utf8.RuneCountInString(summary))
	i := 0

	for _, r := range summary {
		// If our utf8 grapheme cannot be encoded in a single byte, skip.
		if utf8.RuneLen(r) != 1 {
			continue // 👋
		}

		if unicode.IsPunct(r) {
			continue
		}

		// If we're not a letter, number, or even printable, or we're
		// a space char, convert to hyphen.
		if (!unicode.IsLetter(r) && !unicode.IsNumber(r)) || unicode.IsSpace(r) {
			r = rune('-')
			// Do not allow two "-" hyphens in a row
			if i > 0 {
				if truncated[i-1] == rune('-') {
					continue
				}
			} else {
				continue
			}
		}

		truncated[i] = r

		if i > 20 {
			break
		}

		i++
	}

	truncated = truncated[:i]

	loweredWithID := strings.ToLower(fmt.Sprintf("%v-%s", id, string(truncated)))

	return fmt.Sprintf("nib.*.%s.%s", loweredWithID, ext)
}

func MustEditBytes(data []byte, tmpFilename string) []byte {
	editor := strings.Fields(os.Getenv("EDITOR"))

	if len(editor) == 0 {
		editor = []string{"vim"}
	}

	tmpfile, err := os.CreateTemp("", tmpFilename)
	if err != nil {
		ExitFail("Could not create temporary file to edit")
	}

	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file: %v\n", err)
		}
	}()

	_, err = tmpfile.Write(data)
	if err != nil {
		ExitFail("Could not write to temporary file to edit")
	}

	if err := tmpfile.Close(); err != nil {
		ExitFail("Could not close temporary file to edit")
	}

	err = RunCmd(editor[0], append(editor[1:], tmpfile.Name())...)
	if err != nil {
		ExitFail("Failed to run $EDITOR")
	}

	data, err = os.ReadFile(tmpfile.Name())
	if err != nil {
		ExitFail("Could not read back temporary edited file")
	}

	return data
}

func StrSliceContains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

// generics pls...
func IntSliceContains(haystack []int, needle int) bool {
	return slices.Contains(haystack, needle)
}

func StrSliceContainsAll(subset, superset []string) bool {
	for _, have := range subset {
		foundInSuperset := slices.Contains(superset, have)
		if !foundInSuperset {
			return false
		}
	}

	return true
}

func IsValidStateTransition(from string, to string) bool {
	for _, transition := range VALID_STATUS_TRANSITIONS {
		if from == transition[0] && to == transition[1] {
			return true
		}
	}

	return false
}

func MustOpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		ExitFail("unsupported platform")
	}

	if err != nil {
		ExitFail("Failed to open browser")
	}
}

func DeduplicateStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0

	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		s[j] = v
		j++
	}

	return s[:j]
}

// TerminalWidth is the width tables are rendered at. $COLUMNS is honoured when
// it holds a positive number, otherwise the maximum readable width is used.
// Stdout is never inspected, so redirecting output cannot change the result.
func TerminalWidth() int {
	if columns, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && columns > 0 {
		return columns
	}

	return TABLE_MAX_WIDTH
}

func WriteStdout(data []byte) error {
	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}

	return nil
}
