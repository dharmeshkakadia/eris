No win support, no automated tests.

put this in cpp-ethereum root, to replace its regular CMakeLists.txt, then do this:

```
mkdir build
cd build
cmake ..
make
```

This would build all compilers, and be equal to running: `cmake .. -DLLL=1, -DSOLIDITY=1, -DSERPENT=1`

to only build a subset, only include the flags. To build only LLL: `cmake .. -DLLL=1`
