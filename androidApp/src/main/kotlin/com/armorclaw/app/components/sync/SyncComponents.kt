/**
 * Sync status components for ArmorChat
 *
 * This package provides visual sync status indicators, error handling,
 * and device management components for the ArmorChat application.
 *
 * Key Components:
 * - [SyncStatusBar]: Full sync status bar with visual indicators
 * - [SyncIndicatorCompact]: Compact indicator for headers/toolbars
 * - [SyncStatusChip]: Small chip for lists/cards
 * - [ConnectionErrorBanner]: Persistent error banner
 * - [ErrorToast]: Transient error notifications
 * - [ConnectionErrorScreen]: Full-screen error state
 * - [ErrorDetailsDialog]: Error details dialog
 *
 * ViewModels:
 * - [SyncStatusViewModel]: Manages sync state and retry logic
 * - [DeviceListViewModel]: Manages device list and verification
 *
 * Usage:
 * ```kotlin
 * // In your screen composable
 * val viewModel: SyncStatusViewModel = koinViewModel()
 * val uiState by viewModel.uiState.collectAsStateWithLifecycle()
 *
 * SyncStatusBar(
 *     syncState = uiState.syncState,
 *     lastSyncTime = uiState.lastSyncTime,
 *     queuedMessageCount = uiState.queuedMessageCount,
 *     onRefreshClick = { viewModel.sync() },
 *     onErrorClick = { viewModel.retryWithBackoff() }
 * )
 * ```
 */

// Sync Status Components
@file:JvmName("SyncComponents")

package com.armorclaw.app.components.sync

// Re-export main components
typealias SyncStatus = com.armorclaw.shared.domain.model.SyncState
