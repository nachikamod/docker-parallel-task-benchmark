# Docker Benchmark

This repo is about docker parallel computation benchmarking using golang worker pool and docker sdk.

For benchmarking I have used a 11.2.1 gcc compiler with alpine as base image and a simple program to find power of a number using recursion.

```cpp
#include <iostream>
using namespace std;

long int calculatePower(int, int);

int main()
{
    int base = 10, powerRaised = 20, result;

    result = calculatePower(base, powerRaised);
    cout << base << "^" << powerRaised << " = " << result;

    return 0;
}

long int calculatePower(int base, int powerRaised)
{
    if (powerRaised != 0)
        return (base*calculatePower(base, powerRaised-1));
    else
        return 1;
}
```

For testing, I created a queue of 100 tasks to execute above program,
with following container configuration. Total time column denotes the total time required to execute the queued tasks.

|  CPU  | Allowed CPU Usage (Percentage) | Workers | Total Time (Seconds) |
|:-----:|:------------------------------:|:-------:|:--------------------:|
|   2   |               100              |    1    |      309.661555      |
|   2   |               100              |    2    |      266.093034      |
|   2   |               50               |    2    |      253.008682      |
| **1** |             **100**            |  **2**  |    **213.415781**    |
|   1   |               100              |    4    |       332.11905      |
|   1   |               50               |    4    |      232.565958      |
|   1   |               100              |    8    |      383.783277      |
|   1   |               50               |    8    |      342.300561      |
|   2   |               100              |    8    |      380.190151      |
|   2   |               50               |    8    |      399.167070      |
| **2** |             **50**             |  **4**  |        **242**       |
|   2   |               100              |    4    |      300.613794      |
|   2   |               50               |    8    |      273.697072      |

System config on which these taks executed - 

CPU - Intel Core i5-8300H @ 2.30GHz<br>
RAM - 24GB

Allowed cpu usage for docker engine - 2 Cores