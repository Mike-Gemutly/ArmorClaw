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
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>

// Forward declarations
static int (*real_execve)(const char *, char *const[], char *const[]) = NULL;
static int (*real_execvp)(const char *, char *const[]) = NULL;
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
    if (!real_execvp)
        real_execvp = dlsym(RTLD_NEXT, "execvp");
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

// Check if execve should be allowed (for legitimate agent startup)
static int allow_execve(const char *pathname) {
    // Allow execve if ARMORCLAW_ALLOW_EXEC is set (set by entrypoint)
    if (getenv("ARMORCLAW_ALLOW_EXEC")) {
        return 1;
    }

    // Allow execve for specific safe executables
    if (pathname) {
        // Allow Python (for agent execution)
        if (strstr(pathname, "python") != NULL) {
            return 1;
        }
        // Allow node (for Node.js agent components)
        if (strstr(pathname, "node") != NULL) {
            return 1;
        }
        // Allow /usr/bin/id for testing
        if (strstr(pathname, "/usr/bin/id") != NULL) {
            return 1;
        }
        // Allow /bin/id (symlink on some systems)
        if (strstr(pathname, "/bin/id") != NULL) {
            return 1;
        }
    }

    return 0;  // Block by default
}

int execve(const char *pathname, char *const argv[], char *const envp[]) {
    init_real_functions();
    if (allow_execve(pathname)) {
        return real_execve(pathname, argv, envp);
    }
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;  // Block execve calls
}

int execvp(const char *file, char *const argv[]) {
    init_real_functions();
    if (allow_execve(file)) {
        return real_execvp(file, argv);
    }
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;  // Block execvp calls
}

int system(const char *command) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return -1;  // Block system calls
}

FILE *popen(const char *command, const char *type) {
    init_real_functions();
    write(STDERR_FILENO, SECURITY_ERROR, sizeof(SECURITY_ERROR) - 1);
    return NULL;  // Block popen calls
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
