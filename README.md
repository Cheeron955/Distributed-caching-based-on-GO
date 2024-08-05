The project mainly implements a distributed cache based on GO language, and the main functions are as follows:

1. a concurrent cache elimination algorithm using the LRU mechanism and mutex;

2. using the http standard library to achieve inter-node communication. 3. using consistent hashing and virtual nodes to solve the cache avalanche and data skew phenomenon;

3. using consistent hash and virtual nodes to solve the cache avalanche and data skew phenomenon. 4. using singleflight to effectively avoid the cache avalanche and data skew phenomenon;

4. singleflight is used to effectively avoid cache hit and penetration;

Compile step:
One-click compilation ./run.sh
