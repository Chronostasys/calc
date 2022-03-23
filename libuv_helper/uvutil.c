#define GC_THREADS
#include "uv.h"
#include "gc.h"

uv_timer_t* new_timer2(){
    uv_timer_t t;
    return GC_MALLOC(sizeof t);
}

uv_tcp_t* new_tcp(){
    uv_tcp_t server;
    return GC_MALLOC(sizeof server);
}

struct sockaddr_in* new_addr(){
    struct sockaddr_in server;
    return GC_MALLOC(sizeof server);
}

uv_idle_t* new_idle(){
    uv_idle_t t;
    return GC_MALLOC(sizeof t);
}

uv_write_t* new_write(){
    uv_write_t t;
    return GC_MALLOC(sizeof t);
}
uv_async_t* new_async(){
    uv_async_t t;
    return GC_MALLOC(sizeof t);
}

uv_connect_t* new_conn(){
    uv_connect_t t;
    return GC_malloc_uncollectable(sizeof t);
}

uv_stream_t* get_conn_stream(uv_connect_t* t) {
    return t->handle;
}

void* get_tcp_data(uv_tcp_t* t) {
    return t->data;
}
void set_tcp_data(uv_tcp_t* t, void* data) {
    t->data = data;
    return;
}

void* GC_calloc(size_t count, size_t size){
    return GC_malloc_uncollectable((count)*(size));
}

void replace_allocator() {
    uv_replace_allocator(GC_malloc_uncollectable,GC_realloc,GC_calloc,GC_free);
    return;
}

// #include <stdio.h>
// #include <stdlib.h>
// #include <string.h>
// #include <uv.h>

// #define DEFAULT_PORT 7000
// #define DEFAULT_BACKLOG 128

// uv_loop_t* loop;
// struct sockaddr_in addr;

// typedef struct {
//     uv_write_t req;
//     uv_buf_t buf;
// } write_req_t;

// /// 释放资源的回调函数
// void free_write_req(uv_write_t* req) {
//     write_req_t* wr = (write_req_t*)req;
//     free(wr->buf.base);
//     free(wr);
// }

// ///分配空间存储接受的数据
// void alloc_buffer(uv_handle_t* handle, size_t suggested_size, uv_buf_t* buf) {
//     buf->base = (char*)malloc(suggested_size);
//     buf->len = suggested_size;
// }

// /// 写完成后调用的函数
// /// 释放资源
// void echo_write(uv_write_t* req, int status) {
//     if (status) {
//         fprintf(stderr, "Write error %s\n", uv_strerror(status));
//     }
//     free_write_req(req);
// }

// /// 将从socket套接字读取的数据放入request(req) 然后在写(buf->base nread 个字节)后 调用回调函数检查状态 释放req占用的内存
// /// 这里要注意 正确读取的时候req由回调函数处理
// /// 而EOF/其他错误发生的时候 需要关闭套接字 并释放buf->base所占据的内存
// /// EOF代表套接字已经被关闭
// /// 因为此时没有回调函数
// /// 异步回调很容易出错
// void echo_read(uv_stream_t* client, ssize_t nread, const uv_buf_t* buf) {
//     if (nread > 0) {
//         write_req_t* req = (write_req_t*)malloc(sizeof(write_req_t));
//         req->buf = uv_buf_init(buf->base, nread);
//         uv_write((uv_write_t*)req, client, &req->buf, 1, echo_write);
//         return;
//     }
//     if (nread < 0) {
//         if (nread != UV_EOF)
//             fprintf(stderr, "Read error %s\n", uv_err_name(nread));
//         uv_close((uv_handle_t*)client, NULL);
//     }

//     free(buf->base);
// }

// /// 一个新连接的建立
// void on_new_connection(uv_stream_t* server, int status) {
//     if (status < 0) {
//         fprintf(stderr, "New connection error %s\n", uv_strerror(status));
//         // error!
//         return;
//     }

//     uv_tcp_t* client = (uv_tcp_t*)malloc(sizeof(uv_tcp_t));
//     uv_tcp_init(loop, client);
//     if (uv_accept(server, (uv_stream_t*)client) == 0) {
//         uv_read_start((uv_stream_t*)client, alloc_buffer, echo_read);
//     }
//     else {
//         uv_close((uv_handle_t*)client, NULL);
//     }
// }

// int main() {
//     loop = uv_default_loop();

//     uv_tcp_t server;
//     uv_tcp_init(loop, &server);

//     uv_ip4_addr("0.0.0.0", DEFAULT_PORT, &addr);

//     uv_tcp_bind(&server, (const struct sockaddr*) & addr, 0);
//     int r = uv_listen((uv_stream_t*)&server, DEFAULT_BACKLOG, on_new_connection);
//     if (r) {
//         fprintf(stderr, "Listen error %s\n", uv_strerror(r));
//         return 1;
//     }
//     return uv_run(loop, UV_RUN_DEFAULT);
// }
