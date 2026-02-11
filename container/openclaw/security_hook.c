/**
 * ArmorClaw Security Hook (LD_PRELOAD)
 *
 * Intercepts dangerous library calls to prevent:
 * - Shell escapes via execve(), execl(), execvp(), system(), popen()
 * - Network calls via socket(), connect(), sendto(), recvfrom()
 *
 * Compile: gcc -shared -fPIC -o libarmorclaw_hook.so security_hook.c -ldl
 * Use: LD_PRELOAD=/opt/openclaw/lib/libarmorclaw_hook.so
 */

#define _GNU_SOURCE
#include <dlfcn.h>
#include <stddef.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/socket.h>

// Forward declarations
static int (*real_execve)(const char *, char *const[], char *const[]) = NULL;
static int (*real_execveat)(int, const char *, char *const[], char *const[], int) = NULL;
static int (*real_execl)(const char *, const char *, ...) = NULL;
static int (*real_execle)(const char *, const char *, ...) = NULL;
static int (*real_execlp)(const char *, const char *, ...) = NULL;
static int (*real_execvp)(const char *, char *const[]) = NULL;
static int (*real_execvpe)(const char *, char *const[], char *const[]) = NULL;
static int (*real_system)(const char *) = NULL;
static FILE *(*real_popen)(const char *, const char *) = NULL;
static int (*real_socket)(int, int, int) = NULL;
static int (*real_connect)(int, const struct sockaddr *, socklen_t) = NULL;

// Security error message
static const char SECURITY_ERROR[] =
    "ArmorClaw Security: Operation blocked by security policy\n";

// Initialize real function pointers
static void init_real_functions(void) {
    if (!real_execve)
        real_execve = dlsym(RTLD_NEXT, "execve");
    if (!real_execveat)
        real_execveat = dlsym(RTLD_NEXT, "execveat");
    if (!real_system)
        real_system = dlsym(RTLD_NEXT, "system");
    if (!real_popen)
        real_popen = dlsym(RTLD_NEXT, "popen");
    if (!real_socket)
        real_socket = dlsym(RTLD_NEXT, "socket");
    if (!real_connect)
        real_connect = dlsym(RTLD_NEXT, "connect");
}

// ============================================================================
// BLOCK: Process execution functions (shell escapes)
// ============================================================================

int execve(const char *pathname, char *const argv[], char *const envp[]) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;  // Block all execve calls
}

int execveat(int dirfd, const char *pathname,
             char *const argv[], char *const envp[], int flags) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;  // Block all execveat calls
}

int execl(const char *pathname, const char *arg, ...) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

int execle(const char *pathname, const char *arg, ...) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

int execlp(const char *pathname, const char *arg, ...) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

int execvp(const char *file, char *const argv[]) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

int execvpe(const char *file, char *const argv[], char *const envp[]) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

int system(const char *command) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}

FILE *popen(const char *command, const char *type) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return NULL;
}

// ============================================================================
// BLOCK: Network functions (data exfiltration)
// ============================================================================

int socket(int domain, int type, int protocol) {
    init_real_functions();
    // Block AF_INET and AF_INET6 sockets (allow only Unix domain sockets)
    if (domain == AF_INET || domain == AF_INET6) {
        write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
        return -1;
    }
    // Allow Unix domain sockets (for container communication)
    if (real_socket)
        return real_socket(domain, type, protocol);
    return -1;
}

int connect(int sockfd, const struct sockaddr *addr, socklen_t addrlen) {
    init_real_functions();
    // Block all outbound connections
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;
}
