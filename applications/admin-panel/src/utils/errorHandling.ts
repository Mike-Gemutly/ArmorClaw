/**
 * Error Handling Utilities
 *
 * Provides user-friendly error messages and retry logic for API calls.
 */

export interface AppError {
  code: string;
  message: string;
  recoverable: boolean;
  retryAfter?: number;
}

export const ErrorCodes = {
  // Network errors
  NETWORK_UNAVAILABLE: 'NETWORK_UNAVAILABLE',
  CONNECTION_REFUSED: 'CONNECTION_REFUSED',
  TIMEOUT: 'TIMEOUT',

  // Bridge errors
  BRIDGE_NOT_FOUND: 'BRIDGE_NOT_FOUND',
  BRIDGE_UNAVAILABLE: 'BRIDGE_UNAVAILABLE',
  BRIDGE_VERSION_MISMATCH: 'BRIDGE_VERSION_MISMATCH',

  // Authentication errors
  AUTH_REQUIRED: 'AUTH_REQUIRED',
  AUTH_INVALID: 'AUTH_INVALID',
  SESSION_EXPIRED: 'SESSION_EXPIRED',
  PERMISSION_DENIED: 'PERMISSION_DENIED',

  // Setup errors
  ALREADY_CLAIMED: 'ALREADY_CLAIMED',
  INVALID_PASSPHRASE: 'INVALID_PASSPHRASE',
  DEVICE_NOT_TRUSTED: 'DEVICE_NOT_TRUSTED',

  // Validation errors
  INVALID_INPUT: 'INVALID_INPUT',
  MISSING_FIELD: 'MISSING_FIELD',

  // Server errors
  INTERNAL_ERROR: 'INTERNAL_ERROR',
  RATE_LIMITED: 'RATE_LIMITED',

  // Unknown
  UNKNOWN: 'UNKNOWN',
} as const;

/**
 * Maps raw API errors to user-friendly messages
 */
export function mapError(error: unknown): AppError {
  // Network errors
  if (error instanceof TypeError && error.message.includes('fetch')) {
    return {
      code: ErrorCodes.NETWORK_UNAVAILABLE,
      message: 'Unable to connect to the network. Please check your connection.',
      recoverable: true,
      retryAfter: 3000,
    };
  }

  if (error instanceof Error) {
    const message = error.message.toLowerCase();

    // Connection refused
    if (message.includes('connection refused') || message.includes('econnrefused')) {
      return {
        code: ErrorCodes.CONNECTION_REFUSED,
        message: 'Unable to reach the ArmorClaw bridge. Is it running?',
        recoverable: true,
        retryAfter: 5000,
      };
    }

    // Timeout
    if (message.includes('timeout') || message.includes('etimedout')) {
      return {
        code: ErrorCodes.TIMEOUT,
        message: 'The request took too long. Please try again.',
        recoverable: true,
        retryAfter: 2000,
      };
    }

    // Parse JSON-RPC error
    if (message.includes('jsonrpc') || message.includes('rpc')) {
      return parseRpcError(message);
    }

    // HTTP status errors
    if (message.includes('401') || message.includes('unauthorized')) {
      return {
        code: ErrorCodes.AUTH_REQUIRED,
        message: 'Authentication required. Please log in.',
        recoverable: false,
      };
    }

    if (message.includes('403') || message.includes('forbidden')) {
      return {
        code: ErrorCodes.PERMISSION_DENIED,
        message: 'You do not have permission to perform this action.',
        recoverable: false,
      };
    }

    if (message.includes('429') || message.includes('rate limit')) {
      return {
        code: ErrorCodes.RATE_LIMITED,
        message: 'Too many requests. Please wait a moment.',
        recoverable: true,
        retryAfter: 10000,
      };
    }

    if (message.includes('500') || message.includes('internal')) {
      return {
        code: ErrorCodes.INTERNAL_ERROR,
        message: 'An internal error occurred. Please try again.',
        recoverable: true,
        retryAfter: 3000,
      };
    }
  }

  // Unknown error
  return {
    code: ErrorCodes.UNKNOWN,
    message: 'An unexpected error occurred. Please try again.',
    recoverable: true,
    retryAfter: 3000,
  };
}

/**
 * Parses JSON-RPC specific error messages
 */
function parseRpcError(message: string): AppError {
  if (message.includes('already claimed') || message.includes('admin established')) {
    return {
      code: ErrorCodes.ALREADY_CLAIMED,
      message: 'This device has already been claimed by an administrator.',
      recoverable: false,
    };
  }

  if (message.includes('invalid passphrase') || message.includes('invalid credentials')) {
    return {
      code: ErrorCodes.INVALID_PASSPHRASE,
      message: 'The passphrase you entered is incorrect.',
      recoverable: true,
    };
  }

  if (message.includes('not trusted') || message.includes('untrusted device')) {
    return {
      code: ErrorCodes.DEVICE_NOT_TRUSTED,
      message: 'This device is not trusted. Please contact an administrator.',
      recoverable: false,
    };
  }

  if (message.includes('session') && message.includes('expired')) {
    return {
      code: ErrorCodes.SESSION_EXPIRED,
      message: 'Your session has expired. Please log in again.',
      recoverable: false,
    };
  }

  if (message.includes('invalid') && message.includes('input')) {
    return {
      code: ErrorCodes.INVALID_INPUT,
      message: 'Invalid input provided. Please check your entries.',
      recoverable: true,
    };
  }

  return {
    code: ErrorCodes.UNKNOWN,
    message: 'An error occurred while communicating with the bridge.',
    recoverable: true,
    retryAfter: 3000,
  };
}

/**
 * Retry wrapper with exponential backoff
 */
export async function withRetry<T>(
  fn: () => Promise<T>,
  options: {
    maxRetries?: number;
    baseDelay?: number;
    maxDelay?: number;
    shouldRetry?: (error: AppError) => boolean;
  } = {}
): Promise<T> {
  const {
    maxRetries = 3,
    baseDelay = 1000,
    maxDelay = 30000,
    shouldRetry = (error) => error.recoverable,
  } = options;

  let lastError: AppError | null = null;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = mapError(error);

      if (attempt === maxRetries || !shouldRetry(lastError)) {
        throw lastError;
      }

      const delay = Math.min(
        lastError.retryAfter || baseDelay * Math.pow(2, attempt),
        maxDelay
      );

      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}

/**
 * Error boundary fallback props type
 */
export interface ErrorFallbackProps {
  error: AppError;
  resetError: () => void;
}
