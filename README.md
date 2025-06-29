# 🧾 `check_backup`: Because Your Backups Deserve Attention

Tired of wondering if your backups are still alive? This tool pokes them with a stick — and tells Nagios whether to chill or panic.
It checks:

* **Is your latest backup fresh enough?**
* **Is it big enough to be real?**
* **Is there enough disk space left for the next hundred?**

If not — it complains *loudly*. Like a good sysadmin should.

---

## 📦 Installation (copy-paste friendly)

> No root? No problem. Just copy to `/usr/local/bin` or somewhere in your `$PATH`.

### 🐧 Linux

#### x86-64 (Intel/AMD)

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_linux_amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### ARM64 (Raspberry Pi 4+, Apple M1 servers)

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_linux_arm64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### ARMv7 (legacy Pis, embedded)

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_linux_arm \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

### 🍏 macOS

#### Intel Macs

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_darwin_amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### Apple Silicon (M1/M2/M3)

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_darwin_arm64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

### 🐡 BSD (yes, even those)

#### FreeBSD

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_freebsd_amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### OpenBSD

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_openbsd_amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

#### NetBSD

```bash
curl -L https://github.com/matveynator/check_backup/releases/download/latest/check_backup_netbsd_amd64 \
  -o /usr/local/bin/check_backup && chmod +x /usr/local/bin/check_backup
```

---

## 🧪 Example Check

```bash
check_backup \
  -d /backup \
  -p "*.tar.gz" \
  -c 86400 \
  -s 10485760
```

### Sample Output

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

## 🔌 Nagios Integration

Make it scream when backups go stale.

**Command definition:**

```cfg
define command {
  command_name    check_backup
  command_line    /usr/local/bin/check_backup -d $ARG1$ -p $ARG2$ -c $ARG3$ -s $ARG4$
}
```

**Service example:**

```cfg
define service {
  use                 generic-service
  host_name           backup-server
  service_description Check backup freshness and space
  check_command       check_backup!/backup!"*.tar.gz"!86400!10485760
}
```

---

Backups are boring — until they're not.
Make sure someone (or something) is watching them. 👀
Happy monitoring!
