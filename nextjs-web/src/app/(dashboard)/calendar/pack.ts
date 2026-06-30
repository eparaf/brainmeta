// Column-packing for overlapping events, the same idea Google/Apple Calendar use:
// group events that overlap in time into a cluster, then lay them out side by side.

export type Span = { start: number; end: number }; // minutes from midnight

export type Placed<T> = { item: T; col: number; cols: number };

export function packDay<T extends Span>(items: T[]): Placed<T>[] {
  const sorted = [...items].sort((a, b) => a.start - b.start || a.end - b.end);
  const placed: Placed<T>[] = [];
  let cluster: Placed<T>[] = [];
  let clusterEnd = -1;

  const flush = () => {
    const cols = cluster.reduce((m, p) => Math.max(m, p.col + 1), 0);
    for (const p of cluster) p.cols = cols;
    placed.push(...cluster);
    cluster = [];
    clusterEnd = -1;
  };

  for (const item of sorted) {
    if (cluster.length && item.start >= clusterEnd) flush();
    // Find the first free column within the active cluster.
    const taken = new Set(cluster.filter((p) => p.item.end > item.start).map((p) => p.col));
    let col = 0;
    while (taken.has(col)) col++;
    cluster.push({ item, col, cols: 1 });
    clusterEnd = Math.max(clusterEnd, item.end);
  }
  if (cluster.length) flush();
  return placed;
}
