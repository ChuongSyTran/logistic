#!/bin/bash
set -e

mysql -u"root" -p"$MYSQL_ROOT_PASSWORD" <<-EOSQL
  CREATE USER IF NOT EXISTS '${REPLICATION_USER}'@'%'
    IDENTIFIED WITH caching_sha2_password BY '${REPLICATION_PASSWORD}';
  GRANT REPLICATION SLAVE ON *.* TO '${REPLICATION_USER}'@'%';
  FLUSH PRIVILEGES;
EOSQL

echo "[init-master] đã tạo user replication '${REPLICATION_USER}'"
