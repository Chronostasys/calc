#include "uvutil.h"

uv_timer_t *new_timer2()
{
    uv_timer_t t;
    return GC_MALLOC(sizeof t);
}

uv_tcp_t *new_tcp()
{
    uv_tcp_t server;
    return GC_MALLOC(sizeof server);
}

struct sockaddr_in *new_addr()
{
    struct sockaddr_in server;
    return GC_MALLOC(sizeof server);
}

uv_idle_t *new_idle()
{
    uv_idle_t t;
    return GC_MALLOC(sizeof t);
}

uv_write_t *new_write()
{
    uv_write_t t;
    return GC_MALLOC(sizeof t);
}
uv_async_t *new_async()
{
    uv_async_t t;
    return GC_MALLOC(sizeof t);
}

uv_connect_t *new_conn()
{
    uv_connect_t t;
    return GC_malloc_uncollectable(sizeof t);
}

uv_stream_t *get_conn_stream(uv_connect_t *t)
{
    return t->handle;
}

void *get_tcp_data(uv_tcp_t *t)
{
    return t->data;
}
void set_tcp_data(uv_tcp_t *t, void *data)
{
    t->data = data;
    return;
}

void *GC_calloc(size_t count, size_t size)
{
    return GC_malloc_uncollectable((count) * (size));
}

void replace_allocator()
{
    uv_replace_allocator(GC_malloc_uncollectable, GC_realloc, GC_calloc, GC_free);
    return;
}

pthread_cond_t *new_pthread_cond_t()
{
    pthread_cond_t t;
    return GC_MALLOC(sizeof t);
}

pthread_mutex_t *new_pthread_mutex_t()
{
    pthread_mutex_t t;
    return GC_MALLOC(sizeof t);
}

void print_uv_err(int err)
{
    printf("%s", uv_err_name(err));
    return;
}

uv_buf_t *new_buf_t()
{
    uv_buf_t t;
    return GC_MALLOC(sizeof t);
}

char *get_buf_data(uv_buf_t *t)
{
    return t->base;
}

ULONG get_buf_len(uv_buf_t *t)
{
    return t->len;
}

void set_buf_data(uv_buf_t *t, char *data)
{
    t->base = data;
    return;
}

void set_buf_len(uv_buf_t *t, ULONG len)
{

    t->len = len;
    return;
}

int get_available_parallelism()
{
    uv_cpu_info_t *cpu_infos;
    int count;
    int err = uv_cpu_info(&cpu_infos, &count);
    if (err)
        return -1;
    uv_free_cpu_info(cpu_infos, count);
    return count;
}
