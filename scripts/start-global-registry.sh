#!/bin/sh

# Automatically determine the primary IP address of this machine
ETCD_IP=$(hostname -I | awk '{print $1}')

# Check if we got an IP
if [ -z "$ETCD_IP" ]; then
    echo "Could not determine local IP address. Please set it manually."
    # Fallback to a default value if needed, or exit
    ETCD_IP="127.0.0.1"
fi

echo "Starting etcd, advertising IP: $ETCD_IP"

docker run -d \
  -p 2379:2379 \
  -p 2380:2380 \
  --name Etcd-server \
  gcr.io/etcd-development/etcd:v3.5.13 \
  /usr/local/bin/etcd \
  --name s1 \
  --data-dir /etcd-data \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://${ETCD_IP}:2379 \
  --listen-peer-urls http://0.0.0.0:2380 \
  --initial-advertise-peer-urls http://${ETCD_IP}:2380 \
  --initial-cluster s1=http://${ETCD_IP}:2380 \
  --initial-cluster-token tkn \
  --initial-cluster-state new \
  --max-request-bytes 52428800 # To allow fat-zip for go runtime
