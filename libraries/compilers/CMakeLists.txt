# cmake global
cmake_minimum_required(VERSION 2.8.12)

project(ethereum)

list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_SOURCE_DIR}/cmake")

function(createBuildInfo)
    # Set build platform; to be written to BuildInfo.h
    set(ETH_BUILD_PLATFORM "${TARGET_PLATFORM}")
    if (CMAKE_COMPILER_IS_GNUCXX)
        set(ETH_BUILD_PLATFORM "${ETH_BUILD_PLATFORM}/g++")
    elseif ("${CMAKE_CXX_COMPILER_ID}" STREQUAL "Clang")
        set(ETH_BUILD_PLATFORM "${ETH_BUILD_PLATFORM}/clang")
    else ()
        set(ETH_BUILD_PLATFORM "${ETH_BUILD_PLATFORM}/unknown")
    endif ()

    # Generate header file containing useful build information
    add_custom_target(BuildInfo.h ALL COMMAND bash ${CMAKE_CURRENT_SOURCE_DIR}/BuildInfo.sh ${CMAKE_CURRENT_SOURCE_DIR} ${CMAKE_CURRENT_BINARY_DIR} ${_cmake_build_type} ${ETH_BUILD_PLATFORM})
    include_directories(${CMAKE_CURRENT_BINARY_DIR})

    set(CMAKE_INCLUDE_CURRENT_DIR ON)
    set(SRC_LIST BuildInfo.h)
endfunction()



######################################################################################################

# relative paths
cmake_policy(SET CMP0015 NEW)

# instead of poking around in the cmake dependency files, just do this for now...
set(HEADLESS 1)

# Default TARGET_PLATFORM to "linux".
set(TARGET_PLATFORM CACHE STRING "linux")
if ("x${TARGET_PLATFORM}" STREQUAL "x")
    set(TARGET_PLATFORM "linux")
endif ()

if ("${TARGET_PLATFORM}" STREQUAL "linux")
    set(CMAKE_THREAD_LIBS_INIT pthread)
endif ()

include(EthCompilerSettings)

message("-- CXXFLAGS: ${CMAKE_CXX_FLAGS}")

string(TOLOWER ${CMAKE_SYSTEM_NAME} _system_name)
set (ETH_DEPENDENCY_INSTALL_DIR "${CMAKE_CURRENT_SOURCE_DIR}/extdep/install/${_system_name}")
set (CMAKE_PREFIX_PATH ${ETH_DEPENDENCY_INSTALL_DIR})

# Dependencies must have a version number, to ensure reproducible build. The version provided here is the one that is in the extdep repository. If you use system libraries, version numbers may be different.

find_package (CryptoPP 5.6.2 EXACT REQUIRED)
message(" - CryptoPP header: ${CRYPTOPP_INCLUDE_DIRS}")
message(" - CryptoPP lib   : ${CRYPTOPP_LIBRARIES}")

find_package (LevelDB REQUIRED)
message(" - LevelDB header: ${LEVELDB_INCLUDE_DIRS}")
message(" - LevelDB lib: ${LEVELDB_LIBRARIES}")

# TODO the Jsoncpp package does not yet check for correct version number
find_package (Jsoncpp 0.60 REQUIRED)
message(" - Jsoncpp header: ${JSONCPP_INCLUDE_DIRS}")
message(" - Jsoncpp lib   : ${JSONCPP_LIBRARIES}")

# TODO readline package does not yet check for correct version number
# TODO make readline package dependent on cmake options
# TODO get rid of -DETH_READLINE
find_package (Readline 6.3.8)
if (READLINE_FOUND)
	message (" - readline header: ${READLINE_INCLUDE_DIRS}")
	message (" - readline lib   : ${READLINE_LIBRARIES}")
	add_definitions(-DETH_READLINE)
endif ()

# TODO gmp package does not yet check for correct version number
# TODO it is also not required in msvc build
find_package (Gmp 6.0.0)
if (GMP_FOUND)
	message(" - gmp Header: ${GMP_INCLUDE_DIRS}")
	message(" - gmp lib   : ${GMP_LIBRARIES}")
endif()

# use multithreaded boost libraries, with -mt suffix
set(Boost_USE_MULTITHREADED ON)

if (APPLE)

# use static boost libraries *.a
	set(Boost_USE_STATIC_LIBS ON) 

elseif (UNIX)
# use dynamic boost libraries .dll
	set(Boost_USE_STATIC_LIBS OFF) 

endif()

find_package(Boost 1.54.0 REQUIRED COMPONENTS thread date_time system regex chrono filesystem unit_test_framework program_options)

message(" - boost header: ${Boost_INCLUDE_DIRS}")
message(" - boost lib   : ${Boost_LIBRARIES}")

if (APPLE)
	link_directories(/usr/local/lib)
	include_directories(/usr/local/include)
endif()

createBuildInfo()

add_subdirectory(libdevcore)
add_subdirectory(libevmcore)

if (LLL)
	add_subdirectory(liblll)
	add_subdirectory(lllc)
endif ()

if (SERPENT)
	add_subdirectory(libserpent)
	add_subdirectory(sc)
endif()

if (SOLIDITY)
	add_subdirectory(libsolidity)
	add_subdirectory(solc)
endif ()

# No flags just build them all
if (NOT LLL AND NOT SERPENT AND NOT SOLIDITY)
	add_subdirectory(liblll)
	add_subdirectory(lllc)
	add_subdirectory(libserpent)
	add_subdirectory(sc)
	add_subdirectory(libsolidity)
	add_subdirectory(solc)
endif ()
