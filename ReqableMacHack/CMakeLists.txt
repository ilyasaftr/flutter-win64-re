cmake_minimum_required(VERSION 3.20)
project(ReqableMacHack LANGUAGES C)

add_library(reqablemachack SHARED
    "${CMAKE_CURRENT_LIST_DIR}/inject.c"
)

set_target_properties(reqablemachack PROPERTIES
    OUTPUT_NAME "reqablemachack"
    PREFIX "lib"
    SUFFIX ".dylib"
)

target_include_directories(reqablemachack PRIVATE "${CMAKE_CURRENT_LIST_DIR}")

# Injector dylib: keep it lean, no extra deps.
if(APPLE)
    target_link_options(reqablemachack PRIVATE "-Wl,-dead_strip")
endif()
