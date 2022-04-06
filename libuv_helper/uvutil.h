#define GC_THREADS
#include <pthread.h>
#include "uv.h"
#include "gc.h"

#define ULONG unsigned long

uv_timer_t* new_timer2();
uv_tcp_t* new_tcp();
struct sockaddr_in* new_addr();
uv_idle_t* new_idle();
uv_write_t* new_write();
uv_async_t* new_async();
uv_connect_t* new_conn();
uv_stream_t* get_conn_stream(uv_connect_t* t);
void* get_tcp_data(uv_tcp_t* t);
void set_tcp_data(uv_tcp_t* t, void* data);
void* GC_calloc(size_t count, size_t size);
void replace_allocator();
pthread_cond_t* new_pthread_cond_t();
pthread_mutex_t* new_pthread_mutex_t();
void print_uv_err(int err);
uv_buf_t* new_buf_t();
char* get_buf_data(uv_buf_t* t);
ULONG get_buf_len(uv_buf_t* t);
void set_buf_data(uv_buf_t* t, char* data);
void set_buf_len(uv_buf_t* t, ULONG len);
int get_available_parallelism();
