import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

# Load data
df = pd.read_csv("experiment_results.csv")

# Convert timestamp to relative time (seconds from start of each policy)
df['timestamp'] = pd.to_datetime(df['timestamp'], unit='s')
df = df.sort_values('timestamp')

# Split data by policy
rr_data = df[df['policy'] == 'RoundRobin'].copy()
mab_data = df[df['policy'] == 'MAB'].copy()

# Reset time to 0 for both for comparison
rr_data['relative_time'] = (rr_data['timestamp'] - rr_data['timestamp'].min()).dt.total_seconds()
mab_data['relative_time'] = (mab_data['timestamp'] - mab_data['timestamp'].min()).dt.total_seconds()

# --- Plot 1: Cumulative Response Time (Global) ---
# Sum of all response times up to point T
plt.figure(figsize=(10, 6))

# Calculate cumulative sum
rr_data['cumsum_time'] = rr_data['response_time_ms'].cumsum()
mab_data['cumsum_time'] = mab_data['response_time_ms'].cumsum()

plt.plot(rr_data['relative_time'], rr_data['cumsum_time'], label='Baseline (Round Robin)', color='gray', linestyle='--')
plt.plot(mab_data['relative_time'], mab_data['cumsum_time'], label='MAB Strategy', color='blue', linewidth=2)

plt.xlabel('Experiment Time (seconds)')
plt.ylabel('Cumulative Response Time (ms)')
plt.title('Global System Throughput: Baseline vs MAB')
plt.legend()
plt.grid(True)
plt.savefig('cumulative_time.png')
plt.show()

# --- Plot 2: Architecture Selection Over Time (MAB only) ---
# To see how it learned
plt.figure(figsize=(12, 5))
sns.scatterplot(data=mab_data, x='relative_time', y='response_time_ms', hue='node_arch', style='function', alpha=0.6)
plt.title('MAB: Architecture Selection & Latency over Time')
plt.xlabel('Time (s)')
plt.ylabel('Latency (ms)')
plt.savefig('mab_learning_scatter.png')
plt.show()