// check_backup.go — Nagios/NRPE backup checker (Unix). 2025 © Matvey
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

/* Nagios states */
const (
	OK = iota
	WARNING
	CRITICAL
	UNKNOWN
)

/* Flags */
var (
	dirsCSV, pattern string
	ctimeMax, minSize int64
	sampleN           int
	warnPct, critPct  = 80.0, 90.0
)
func init() {
	flag.StringVar(&dirsCSV, "d", "",  "Backup directories (comma-separated)  *required*")
	flag.StringVar(&pattern, "p", "*", "Glob pattern (default \"*\")")
	flag.Int64Var(&ctimeMax, "c", 0,  "CRITICAL if newest backup older than N seconds  *required*")
	flag.Int64Var(&minSize, "s", 0,  "CRITICAL if newest backup smaller than N bytes   *required*")
	flag.IntVar(&sampleN,  "n", 10,  "How many recent backups to analyse frequency")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `
check_backup — Nagios/NRPE plugin that validates your backups

Usage:
  check_backup -d DIR1[,DIR2] -p PATTERN -c SECONDS -s BYTES [options]

Required flags
  -d  BACKUP_DIRS   Comma-separated list of backup folders
  -c  SECONDS       Max allowed age of latest backup
  -s  BYTES         Min allowed size of latest backup

Example:
  check_backup -d /backups/db,/backups/files -p "*.tar.gz" -c 86400 -s 10485760
`)
	}
}

/* Helpers */
func human(b int64) string {
	const u = 1024
	if b < u { return fmt.Sprintf("%d B", b) }
	div, exp := int64(u), 0
	for n := b/u; n >= u; n /= u { div *= u; exp++ }
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
func durRound(d time.Duration) string { return d.Round(time.Second).String() }
func daysHours(d time.Duration) string {
	days := int(d.Hours())/24; hrs := int(d.Hours())%24
	return fmt.Sprintf("%dd%dh", days, hrs)
}
/* Turn mean interval into human phrase */
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

/* Disk usage (Unix) */
func diskUsage(path string)(tot, free int64, err error){
	var st syscall.Statfs_t
	if err = syscall.Statfs(path,&st); err!=nil {return}
	tot = int64(st.Blocks)*int64(st.Bsize)
	free= int64(st.Bavail)*int64(st.Bsize); return
}

/* Data structures */
type backup struct{ path string; mt time.Time; size int64 }
type result struct{
	dir string; state int; reason string
	last backup; age float64; avgInt float64
	avgSize, tot, free int64; leftFiles int
	leftTime time.Duration; usedPct float64
}

/* Analyse one directory */
func analyse(d string) result{
	r:=result{dir:d,state:OK,reason:"OK"}
	// Collect matching files
	var list []backup
	filepath.Walk(d,func(p string,i os.FileInfo,e error)error{
		if e!=nil||i.IsDir(){return nil}
		if ok,_:=filepath.Match(pattern,filepath.Base(p));ok{
			list=append(list,backup{p,i.ModTime(),i.Size()})
		}
		return nil
	})
	if len(list)==0{r.state, r.reason=UNKNOWN,"no files";return r}
	sort.Slice(list,func(i,j int)bool{return list[i].mt.After(list[j].mt)})
	r.last=list[0]; r.age=time.Since(r.last.mt).Seconds()
	// Mean interval & size
	sample:=list;if len(sample)>sampleN{sample=sample[:sampleN]}
	var sumInt float64;var sumSz int64
	for i:=range sample{
		sumSz+=sample[i].size
		if i<len(sample)-1{sumInt+=sample[i].mt.Sub(sample[i+1].mt).Seconds()}
	}
	r.avgSize=sumSz/int64(len(sample))
	if len(sample)>1{r.avgInt=sumInt/float64(len(sample)-1)}
	// Disk
	tot,free,err:=diskUsage(d)
	if err!=nil{r.state,r.reason=UNKNOWN,"disk error";return r}
	r.tot, r.free = tot, free
	r.usedPct = 100*float64(tot-free)/float64(tot)
	r.leftFiles = int(float64(free)/float64(r.avgSize))
	if r.avgInt>0{r.leftTime=time.Duration(r.avgInt*float64(r.leftFiles))*time.Second}
	// State classification
	switch{
	case int64(r.age)>=ctimeMax:
		r.state,r.reason=CRITICAL,"backup too old"
	case r.last.size<=minSize:
		r.state,r.reason=CRITICAL,"backup too small"
	case r.usedPct>=critPct:
		r.state,r.reason=CRITICAL,fmt.Sprintf("disk %.1f%% full",r.usedPct)
	case r.usedPct>=warnPct:
		r.state,r.reason=WARNING, fmt.Sprintf("disk %.1f%% full",r.usedPct)
	}
	return r
}

/* MAIN */
func main(){
	flag.Parse()
	if len(os.Args)==1{flag.Usage();os.Exit(UNKNOWN)}
	if dirsCSV==""||ctimeMax==0||minSize==0{
		fmt.Println("UNKNOWN: -d, -c and -s are required. Use -h for help."); os.Exit(UNKNOWN)
	}
	var dirs []string
	for _,v:=range strings.Split(dirsCSV,","){
		if s:=strings.TrimSpace(v);s!=""{dirs=append(dirs,s)}
	}

	var res []result; worst:=OK
	for _,d:=range dirs{
		r:=analyse(d);res=append(res,r)
		if r.state>worst{worst=r.state}
	}

	/* Nagios one-liner */
	stNames:=[]string{"OK","WARNING","CRITICAL","UNKNOWN"}
	fmt.Print(stNames[worst]+":")
	for i,r:=range res{
		if i>0{fmt.Print(",")}
		fmt.Printf(" [%s] %s",r.dir,r.reason)
	}
	fmt.Println()

	/* Verbose report */
	for _,r:=range res{
		start:=r.last.mt.Add(-time.Duration(r.avgInt)*time.Second)
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
			r.age,
			human(r.free), human(r.tot), r.usedPct,
			r.leftFiles, human(r.avgSize),
		)
		if r.avgInt>0{
			fmt.Printf(`
Frequency:      %s
Forecast:       space should last ≈ %s`,
				freqPhrase(r.avgInt),
				daysHours(r.leftTime))
		}else{
			fmt.Print(`
Frequency:      not enough data`)
		}
		fmt.Println("\n")
	}

	os.Exit(worst)
}

