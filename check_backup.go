package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

/* Nagios exit codes */
const (
	OK = iota
	WARNING
	CRITICAL
	UNKNOWN
)

/* CLI flags */
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
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `
check_backup — Nagios/NRPE plugin to validate backup freshness and disk usage

Usage:
  check_backup -d DIR1[,DIR2] -p PATTERN -c SECONDS -s BYTES [options]

Required flags:
  -d  DIRECTORIES   Backup directories (comma-separated)
  -c  SECONDS       Max allowed age of latest backup
  -s  BYTES         Minimum allowed size of latest backup

Example:
  check_backup -d /backups -p "*.tar.gz" -c 86400 -s 10485760
`)
	}
}

/* Structs */
type backup struct {
	path string
	mt   time.Time
	size int64
}

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

/* Disk usage using x/sys/unix (portable) */
func diskUsage(path string) (total, free int64, err error) {
	var st unix.Statfs_t
	if err = unix.Statfs(path, &st); err != nil {
		return
	}
	blockSize := int64(st.Bsize)
	total = int64(st.Blocks) * blockSize
	free = int64(st.Bavail) * blockSize
	return
}

/* Helper functions */
func human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func durPretty(d time.Duration) string {
	return d.Round(time.Second).String()
}

func freqPhrase(sec float64) string {
	switch {
	case sec < 90:
		return fmt.Sprintf("about every %.0f s", sec)
	case sec < 90*60:
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

func daysHours(d time.Duration) string {
	return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
}

/* Main analysis function */
func analyse(dir string) result {
	r := result{dir: dir, state: OK, reason: "OK"}

	var list []backup
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if ok, _ := filepath.Match(pattern, filepath.Base(path)); ok {
			list = append(list, backup{path, info.ModTime(), info.Size()})
		}
		return nil
	})

	if len(list) == 0 {
		r.state, r.reason = UNKNOWN, "no files"
		return r
	}

	sort.Slice(list, func(i, j int) bool { return list[i].mt.After(list[j].mt) })
	r.last = list[0]
	r.ageSec = time.Since(r.last.mt).Seconds()

	sample := list
	if len(sample) > sampleN {
		sample = sample[:sampleN]
	}
	var sumInt float64
	var sumSize int64
	for i := range sample {
		sumSize += sample[i].size
		if i < len(sample)-1 {
			sumInt += sample[i].mt.Sub(sample[i+1].mt).Seconds()
		}
	}
	r.avgSize = sumSize / int64(len(sample))
	if len(sample) > 1 {
		r.avgIntSec = sumInt / float64(len(sample)-1)
	}

	total, free, err := diskUsage(dir)
	if err != nil {
		r.state, r.reason = UNKNOWN, "disk error"
		return r
	}
	r.total, r.free = total, free
	r.usedPct = 100 * float64(total-free) / float64(total)
	r.leftFiles = int(float64(r.free) / float64(r.avgSize))
	if r.avgIntSec > 0 {
		r.leftTime = time.Duration(r.avgIntSec*float64(r.leftFiles)) * time.Second
	}

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

/* Main entry */
func main() {
	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(UNKNOWN)
	}

	if dirsCSV == "" || ctimeMax == 0 || minSize == 0 {
		fmt.Println("UNKNOWN: flags -d, -c and -s are required. Use -h for help.")
		os.Exit(UNKNOWN)
	}

	var dirs []string
	for _, s := range strings.Split(dirsCSV, ",") {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			dirs = append(dirs, trimmed)
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

	// Nagios summary line
	states := []string{"OK", "WARNING", "CRITICAL", "UNKNOWN"}
	fmt.Print(states[worst] + ":")
	for i, r := range results {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf(" [%s] %s", r.dir, r.reason)
	}
	fmt.Println()

	// Human-readable output
	for _, r := range results {
		start := r.last.mt.Add(-time.Duration(r.avgIntSec) * time.Second)
		fmt.Printf(`
Newest backup:  %s
Size:           %s
Written:        %s → %s
Elapsed:        %.0f s

Disk:           %s free / %s total (%.1f%% used)
Capacity:       ≈ %d backups (%s each)`,
			r.last.path,
			human(r.last.size),
			start.Format("2006-01-02 15:04:05"),
			r.last.mt.Format("2006-01-02 15:04:05"),
			r.ageSec,
			human(r.free), human(r.total), r.usedPct,
			r.leftFiles, human(r.avgSize),
		)
		if r.avgIntSec > 0 {
			fmt.Printf(`
Frequency:      %s
Forecast:       space should last ≈ %s`,
				freqPhrase(r.avgIntSec),
				daysHours(r.leftTime),
			)
		} else {
			fmt.Print(`
Frequency:      not enough data`)
		}
		fmt.Println("\n")
	}

	os.Exit(worst)
}

