#include "uv.h"
#include "gc.h"

void callback(uv_timer_t* timer){
    return;
}
uv_timer_t* new_timer(){
    uv_timer_t t;
    return GC_MALLOC(sizeof t);
}


