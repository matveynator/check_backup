package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

/* Nagios exit codes */
const (
	OK = iota
	WARNING
	CRITICAL
	UNKNOWN
)

/* Flags */
var (
	dirsCSV, pattern      string
	ctimeMax, minSize     int64
	sampleN               int
	warnPct, critPct      = 80.0, 90.0
)

func init() {
	flag.StringVar(&dirsCSV, "d", "", "Backup directories (comma-separated)  *required*")
	flag.StringVar(&pattern, "p", "*", "Glob pattern (default \"*\")")
	flag.Int64Var(&ctimeMax, "c", 0, "CRITICAL if newest backup older than N seconds  *required*")
	flag.Int64Var(&minSize, "s", 0, "CRITICAL if newest backup smaller than N bytes   *required*")
	flag.IntVar(&sampleN, "n", 10, "How many recent backups to analyse frequency")
}

/* Helpers */
func human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b/unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
func freqPhrase(sec float64) string {
	switch {
	case sec < 90:
		return fmt.Sprintf("about every %.0f s", sec)
	case sec < 5400:
		return fmt.Sprintf("about every %.0f min", sec/60)
	case sec < 3*3600:
		return "about once an hour"
	case sec < 22*3600:
		return fmt.Sprintf("roughly every %.0f h", sec/3600)
	case sec < 36*3600:
		return "about once a day"
	case sec < 7*24*3600:
		return fmt.Sprintf("every %.0f days", sec/86400)
	default:
		return fmt.Sprintf("every %.1f days", sec/86400)
	}
}
func dh(d time.Duration) string { return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24) }
func autoGlob(p string) string {
	if strings.ContainsAny(p, "*?[") {
		return p
	}
	return "*" + p + "*"
}

/* Data structs */
type backup struct{ path string; mt time.Time; size int64 }
type result struct {
	dir                                string
	state                              int
	reason                             string
	last                               backup
	ageSec, avgIntSec, usedPct         float64
	avgSize, total, free               int64
	leftFiles                          int
	leftTime                           time.Duration
}

/* Analyse one directory */
func analyse(dir string) result {
	r := result{dir: dir, state: OK, reason: "OK"}

	/* collect files */
	var files []backup
	filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
		if err != nil || i.IsDir() {
			return nil
		}
		ok, _ := filepath.Match(pattern, filepath.Base(p))
		if ok {
			files = append(files, backup{p, i.ModTime(), i.Size()})
		}
		return nil
	})
	if len(files) == 0 {
		r.state, r.reason = UNKNOWN, "no files"
		return r
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mt.After(files[j].mt) })
	r.last = files[0]
	r.ageSec = time.Since(r.last.mt).Seconds()

	/* averages */
	sample := files
	if len(sample) > sampleN {
		sample = sample[:sampleN]
	}
	var sumInt float64
	var sumSz int64
	for i := range sample {
		sumSz += sample[i].size
		if i < len(sample)-1 {
			sumInt += sample[i].mt.Sub(sample[i+1].mt).Seconds()
		}
	}
	r.avgSize = sumSz / int64(len(sample))
	if len(sample) > 1 {
		r.avgIntSec = sumInt / float64(len(sample)-1)
	}

	/* disk stats */
	tot, free, err := diskUsage(dir)
	if err != nil {
		r.state, r.reason = UNKNOWN, "disk error"
		return r
	}
	r.total, r.free = tot, free
	r.usedPct = 100 * float64(tot-free) / float64(tot)
	r.leftFiles = int(float64(free) / float64(r.avgSize))
	if r.avgIntSec > 0 {
		r.leftTime = time.Duration(r.avgIntSec*float64(r.leftFiles)) * time.Second
	}

	/* state */
	switch {
	case int64(r.ageSec) >= ctimeMax:
		r.state, r.reason = CRITICAL, "backup too old"
	case r.last.size <= minSize:
		r.state, r.reason = CRITICAL, "backup too small"
	case r.usedPct >= critPct:
		r.state, r.reason = CRITICAL, fmt.Sprintf("disk %.1f%% full", r.usedPct)
	case r.usedPct >= warnPct:
		r.state, r.reason = WARNING, fmt.Sprintf("disk %.1f%% full", r.usedPct)
	}
	return r
}

/* MAIN */
func main() {
	flag.Parse()

	if dirsCSV == "" || ctimeMax == 0 || minSize == 0 {
		fmt.Println("UNKNOWN: -d, -c, -s are required. Use -h for help.")
		os.Exit(UNKNOWN)
	}

	pattern = autoGlob(pattern)

	var dirs []string
	for _, p := range strings.Split(dirsCSV, ",") {
		if s := strings.TrimSpace(p); s != "" {
			dirs = append(dirs, s)
		}
	}

	var results []result
	worst := OK
	for _, d := range dirs {
		res := analyse(d)
		results = append(results, res)
		if res.state > worst {
			worst = res.state
		}
	}

	/* Nagios one-liner */
	stateTxt := []string{"OK", "WARNING", "CRITICAL", "UNKNOWN"}[worst]
	fmt.Print(stateTxt, ":")
	for i, r := range results {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf(" [%s] %s", r.dir, r.reason)
	}
	fmt.Println()

	/* Detailed section */
	for _, r := range results {
		if r.state == UNKNOWN && r.reason == "no files" {
			fmt.Printf("\nDirectory %s — no matching files found\n\n", r.dir)
			continue
		}

		fmt.Printf(`
Newest backup:  %s
Size:           %s
Written:        %s
Elapsed:        %s (%.0f s)

Disk:           %s free / %s total (%.1f %% used)
Capacity:       ≈ %d backups (%s each)`,
			r.last.path,
			human(r.last.size),
			r.last.mt.Format("2006-01-02 15:04:05"),
			dh(time.Duration(r.ageSec)*time.Second),
			r.ageSec,
			human(r.free), human(r.total), r.usedPct,
			r.leftFiles, human(r.avgSize),
		)
		if r.avgIntSec > 0 {
			fmt.Printf(`
Frequency:      %s
Forecast:       space should last ≈ %d days`,
				freqPhrase(r.avgIntSec),
				int(r.leftTime.Hours())/24,
			)
		} else {
			fmt.Print(`
Frequency:      not enough data`)
		}
		fmt.Println("\n")
	}

	os.Exit(worst)
}

