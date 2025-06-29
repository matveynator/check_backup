
### 🐧 Linux

#### Linux x86-64 (Intel/AMD servers)

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-linux-amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### Linux arm64 (new Raspberry Pi / ARM servers)

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-linux-arm64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### Linux armv7 (older Pi / embedded)

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-linux-armv7 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

### 🍎 macOS

#### macOS Intel

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-darwin-amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### macOS Apple Silicon (M1/M2/M3)

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-darwin-arm64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

### 🐡 BSD Systems

#### FreeBSD (x86-64)

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-freebsd-amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### OpenBSD

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-openbsd-amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### NetBSD

```bash
curl -L https://github.com/matveynator/check_backup/releases/latest/download/check_backup-netbsd-amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

### 🧪 Example check

```bash
check_backup \
  -d /backup \
  -p "*.tar.gz" \
  -c 86400 \
  -s 10485760
```

**Output:**

```
OK: [/backup]

Newest backup:  /backup/db-20250629_1200.tar.gz
Size:           128.7 MiB
Written:        2025-06-29 12:00:12 → 2025-06-29 12:00:28
Elapsed:        3600 s

Disk:           257.5 GiB free / 1.0 TiB total (74.3 % used)
Capacity:       ≈ 2056 backups (128.7 MiB each)
Frequency:      about once a day
Forecast:       space should last ≈ 2056d0h
```

---

### 🔌 Nagios Integration

**Command definition:**

```cfg
define command {
  command_name    check_backup
  command_line    /usr/local/bin/check_backup -d $ARG1$ -p $ARG2$ -c $ARG3$ -s $ARG4$
}
```

**Service:**

```cfg
define service {
  use                 generic-service
  host_name           backup-server
  service_description Check backup freshness and space
  check_command       check_backup!/backup!"*.tar.gz"!86400!10485760
}
```

