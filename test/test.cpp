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