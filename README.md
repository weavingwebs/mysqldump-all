## Build

```bash
go build
```

## Examples

### Migration from metal to docker

```bash
# Dump data.
mysqldump-all dump ~/backups

# Import into docker.
mysqldump-all import --docker="mysql" ~/backups
```

### DR Backup.

```bash
# Backup without locking certain DBs.
mysqldump-all dump --docker="mysql" --no-locks="big_db_1,big_db_2" ~/backups

# Restore everything, including users (not recommended if restoring to a different MySQL version).
mysqldump-all import --docker="mysql" --include-mysql ~/backups
```
