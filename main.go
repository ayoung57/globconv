package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"globconv/convert"
)

func main() {
	from := flag.String("from", "", "source format: gitignore or rsync")
	to := flag.String("to", "", "target format: gitignore or rsync")
	lenient := flag.Bool("lenient", false, "best-effort convert patterns that don't map cleanly, with warnings on stderr, instead of failing")
	flag.Parse()

	if !validFormat(*from) || !validFormat(*to) || *from == *to {
		fmt.Fprintln(os.Stderr, "globconv: --from and --to are required and must be different, each one of \"gitignore\" or \"rsync\"")
		os.Exit(2)
	}

	lines, err := readLines(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "globconv:", err)
		os.Exit(1)
	}

	opts := convert.Options{Lenient: *lenient}

	var out []string
	var warnings []convert.Warning
	if *from == "gitignore" {
		out, warnings, err = convert.GitignoreToRsync(lines, opts)
	} else {
		out, warnings, err = convert.RsyncToGitignore(lines, opts)
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "globconv: line %d: %s\n", w.Line, w.Message)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "globconv:", err)
		fmt.Fprintln(os.Stderr, "globconv: pass --lenient to convert anyway")
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, line := range out {
		fmt.Fprintln(w, line)
	}
}

func validFormat(f string) bool {
	return f == "gitignore" || f == "rsync"
}

func readLines(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return scanLines(os.Stdin)
	}
	var all []string
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		lines, err := scanLines(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		all = append(all, lines...)
	}
	return all, nil
}

func scanLines(r io.Reader) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
