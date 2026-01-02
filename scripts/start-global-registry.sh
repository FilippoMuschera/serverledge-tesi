#!/bin/sh

# Automatically find Internal IP
INTERNAL_IP=$(hostname -I | awk '{print $1}')

# Find External IP (via Google Cloud metadata server)
PUBLIC_IP=$(curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip)

# Fallback if not on GCP or curl fails
if [ -z "$PUBLIC_IP" ]; then
    echo "Could not determine Public IP. Using Internal as fallback."
    PUBLIC_IP=$INTERNAL_IP
fi

echo "Internal IP: $INTERNAL_IP"
echo "Public IP:   $PUBLIC_IP"


docker run -d \
  -p 2379:2379 \
  -p 2380:2380 \
  --name Etcd-server \
  gcr.io/etcd-development/etcd:v3.5.13 \
  /usr/local/bin/etcd \
  --name s1 \
  --data-dir /etcd-data \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://${INTERNAL_IP}:2379,http://${PUBLIC_IP}:2379 \
  --listen-peer-urls http://0.0.0.0:2380 \
  --initial-advertise-peer-urls http://${INTERNAL_IP}:2380 \
  --initial-cluster s1=http://${INTERNAL_IP}:2380 \
  --initial-cluster-token tkn \
  --initial-cluster-state new