/**
 * Error Handling Utilities for Setup Wizard
 *
 * Provides user-friendly error messages during the setup flow.
 */

export interface SetupError {
  code: string;
  title: string;
  message: string;
  recoverable: boolean;
  retryAfter?: number;
  helpUrl?: string;
}

export const ErrorCodes = {
  BRIDGE_NOT_FOUND: 'BRIDGE_NOT_FOUND',
  CONNECTION_FAILED: 'CONNECTION_FAILED',
  ALREADY_CLAIMED: 'ALREADY_CLAIMED',
  INVALID_PASSPHRASE: 'INVALID_PASSPHRASE',
  SETUP_INCOMPLETE: 'SETUP_INCOMPLETE',
  VALIDATION_ERROR: 'VALIDATION_ERROR',
  NETWORK_TIMEOUT: 'NETWORK_TIMEOUT',
  UNKNOWN: 'UNKNOWN',
} as const;

/**
 * Maps errors to user-friendly setup messages
 */
export function mapSetupError(error: unknown): SetupError {
  if (error instanceof TypeError && error.message.includes('fetch')) {
    return {
      code: ErrorCodes.BRIDGE_NOT_FOUND,
      title: 'Bridge Not Found',
      message: 'The ArmorClaw bridge could not be reached. Make sure it is running on this device.',
      recoverable: true,
      retryAfter: 3000,
    };
  }

  if (error instanceof Error) {
    const msg = error.message.toLowerCase();

    if (msg.includes('already claimed') || msg.includes('admin established')) {
      return {
        code: ErrorCodes.ALREADY_CLAIMED,
        title: 'Device Already Claimed',
        message: 'This device has already been set up by an administrator. Contact them for access.',
        recoverable: false,
      };
    }

    if (msg.includes('invalid passphrase') || msg.includes('invalid credentials')) {
      return {
        code: ErrorCodes.INVALID_PASSPHRASE,
        title: 'Invalid Passphrase',
        message: 'The passphrase you entered is incorrect. Please try again.',
        recoverable: true,
      };
    }

    if (msg.includes('timeout') || msg.includes('etimedout')) {
      return {
        code: ErrorCodes.NETWORK_TIMEOUT,
        title: 'Connection Timeout',
        message: 'The connection took too long. Check your network and try again.',
        recoverable: true,
        retryAfter: 2000,
      };
    }

    if (msg.includes('validation') || msg.includes('invalid input')) {
      return {
        code: ErrorCodes.VALIDATION_ERROR,
        title: 'Invalid Input',
        message: 'Please check your entries and try again.',
        recoverable: true,
      };
    }

    if (msg.includes('connection refused') || msg.includes('econnrefused')) {
      return {
        code: ErrorCodes.CONNECTION_FAILED,
        title: 'Connection Failed',
        message: 'Could not connect to the bridge. Verify the bridge service is running.',
        recoverable: true,
        retryAfter: 5000,
      };
    }
  }

  return {
    code: ErrorCodes.UNKNOWN,
    title: 'Setup Error',
    message: 'An unexpected error occurred during setup. Please try again.',
    recoverable: true,
    retryAfter: 3000,
  };
}

/**
 * Retry with exponential backoff
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  options: {
    maxAttempts?: number;
    baseDelay?: number;
    onRetry?: (attempt: number, error: SetupError) => void;
  } = {}
): Promise<T> {
  const { maxAttempts = 3, baseDelay = 1000, onRetry } = options;

  let lastError: SetupError | null = null;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = mapSetupError(error);

      if (attempt === maxAttempts || !lastError.recoverable) {
        throw lastError;
      }

      onRetry?.(attempt, lastError);

      const delay = lastError.retryAfter || baseDelay * Math.pow(2, attempt - 1);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}
