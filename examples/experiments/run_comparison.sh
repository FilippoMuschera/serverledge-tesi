#!/bin/bash

LOCUST_DURATION="5m"
USERS=2
SPAWN_RATE=2
RESULT_FILE="experiment_results.csv"

rm -f $RESULT_FILE

echo "============================================="
echo "STARTING MAB EXPERIMENT"
echo "============================================="


nohup ~/serverledge-tesi/bin/lb lb-config-MAB.yaml > lb_mab.log 2>&1 &
LB_PID=$!

echo "Load Balancer started (PID: $LB_PID). Waiting for initialization..."
sleep 3

echo "Running Locust for $LOCUST_DURATION..."
export LB_POLICY="MAB_LinUCB"
locust -f locustfile.py \
    --headless \
    --users $USERS \
    --spawn-rate $SPAWN_RATE \
    --run-time $LOCUST_DURATION \
    --host http://localhost:1323

echo "Stopping Load Balancer..."
kill $LB_PID
sleep 5

echo ""
echo "###########################################################"
echo "#                   OPERATIONAL PAUSE                     #"
echo "###########################################################"
echo "Phase 1 has ended."
echo ""
echo "NOW REBOOT BOTH X86 AND ARM TARGET NODES"
echo "Otherwise, the RR will start with warm containers, and this will mean an uneven comparison"
echo ""
read -p "PRESS [ENTER] WHEN YOU ARE READY FOR PHASE 2 (RR)..."
echo ""

echo "============================================="
echo "STARTING RoundRobin EXPERIMENT"
echo "============================================="

nohup ~/serverledge-tesi/bin/lb lb-config-RR.yaml > lb_rr.log 2>&1 &
LB_PID=$!

echo "Load Balancer started (PID: $LB_PID). Waiting for initialization..."
sleep 3

echo "Running Locust for $LOCUST_DURATION..."
export LB_POLICY="RoundRobin"
locust -f locustfile.py \
    --headless \
    --users $USERS \
    --spawn-rate $SPAWN_RATE \
    --run-time $LOCUST_DURATION \
    --host http://localhost:1323

kill $LB_PID

echo "Experiments completed. Data saved to $RESULT_FILE"