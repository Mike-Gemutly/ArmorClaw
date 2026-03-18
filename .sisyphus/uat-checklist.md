# UAT Checklist - OMO Integration

> **Version**: 1.0
> **Last Updated**: 2026-03-15
> **Status**: Ready for Testing

---

## Overview

This checklist covers User Acceptance Testing for the OMO (OhMyOpenagent) Integration features in ArmorChat.

---

## 1. Agent Workspace

### Split View Layout
- [ ] **Desktop/Large Screen**: Split view displays correctly with ActivityLog in right pane
- [ ] **Mobile/Small Screen**: ActivityLog collapses to BottomSheet
- [ ] **Resize Handling**: Pane ratio adjusts smoothly on window resize
- [ ] **Pane Scrolling**: Each pane scrolls independently
- [ ] **Drag Handle**: Drag handle appears and functions for pane resizing

### Activity Log
- [ ] **Timeline Display**: Events appear in chronological order
- [ ] **Event Icons**: Correct icons for each event type (message, task, approval, error)
- [ ] **Timestamp Format**: Human-readable relative timestamps (e.g., "2 min ago")
- [ ] **Event Details**: Expandable details show full event information
- [ ] **Auto-scroll**: New events automatically scroll into view
- [ ] **Empty State**: Appropriate message when no events exist

### Command Bar
- [ ] **Chip Display**: All command chips visible and clickable
- [ ] **Status Chip**: Injects `!status` command
- [ ] **Screenshot Chip**: Injects `!screenshot` command
- [ ] **Stop Chip**: Injects `!stop` command
- [ ] **Pause Chip**: Injects `!pause` command
- [ ] **Logs Chip**: Injects `!logs` command
- [ ] **Text Input**: Command text field accepts input
- [ ] **Submit**: Enter key or send button submits command
- [ ] **Chip Injection**: Clicking chip injects command into text field

---

## 2. Vault Screen

### PII Management
- [ ] **Category Display**: All 12 categories visible (including 6 OMO categories)
- [ ] **Key Listing**: Keys display with name, category, sensitivity
- [ ] **Add Key**: FAB button opens add dialog
- [ ] **Edit Key**: Click on key opens edit dialog
- [ ] **Delete Key**: Long press or swipe to delete
- [ ] **Biometric Auth**: Edit/delete operations require biometric authentication

### OMO Categories
- [ ] **OMO Credentials**: Keys under `OMO_CREDENTIALS` category
- [ ] **OMO Identity**: Keys under `OMO_IDENTITY` category
- [ ] **OMO Settings**: Keys under `OMO_SETTINGS` category
- [ ] **OMO Tokens**: Keys under `OMO_TOKENS` category
- [ ] **OMO Workspace**: Keys under `OMO_WORKSPACE` category
- [ ] **OMO Tasks**: Keys under `OMO_TASKS` category

---

## 3. Agent Studio

### 4-Step Wizard
- [ ] **Step 1 - Role Definition**: Name, type, description inputs work
- [ ] **Step 2 - Skill Selection**: Skills can be selected/deselected
- [ ] **Step 3 - Workflow Builder**: Blockly interface loads and displays
- [ ] **Step 4 - Permissions**: Permission toggles for email, contacts, calendar
- [ ] **Navigation**: Back/Next buttons function correctly
- [ ] **Progress Indicator**: Shows current step (1-4)

### Role Definition (Step 1)
- [ ] **Agent Name**: Text field accepts input
- [ ] **Agent Type**: Dropdown with options (Assistant, Monitor, Automation, Custom)
- [ ] **Description**: Multi-line text input
- [ ] **Validation**: Cannot proceed without required fields

### Skill Selection (Step 2)
- [ ] **Skill Chips**: Display available skills
- [ ] **Selection**: Clicking toggles skill selection
- [ ] **Visual Feedback**: Selected skills highlighted
- [ ] **Categories**: Skills organized by category

### Workflow Builder (Step 3)
- [ ] **Blockly Load**: Blockly WebView loads successfully
- [ ] **Block Palette**: Block categories visible in toolbox
- [ ] **Drag & Drop**: Blocks can be dragged to workspace
- [ ] **Block Connection**: Blocks connect with valid type checking
- [ ] **Workspace Save**: Workspace XML saves correctly

### Permissions (Step 4)
- [ ] **Permission Toggles**: Email, Contacts, Calendar toggles
- [ ] **Sensitivity Level**: Dropdown for sensitivity selection
- [ ] **Create Button**: Creates agent with all collected data
- [ ] **Success Feedback**: Confirmation after agent creation

---

## 4. Navigation

### Route Access
- [ ] **Vault Screen**: Accessible via VAULT route
- [ ] **Agent Studio**: Accessible via AGENT_STUDIO route
- [ ] **Chat with Workspace**: Chat screen includes split view

### Deep Links
- [ ] **Room Deep Link**: Opens specific chat room
- [ ] **Agent Deep Link**: Opens specific agent (if implemented)

---

## 5. Security

### Biometric Authentication
- [ ] **Prompt Display**: Biometric prompt appears for sensitive operations
- [ ] **Success Flow**: Successful auth allows operation
- [ ] **Failure Handling**: Failed auth shows appropriate error
- [ ] **Fallback**: Password/PIN fallback available

### Encryption
- [ ] **Data at Rest**: Vault data encrypted in database
- [ ] **Key Storage**: Keys stored in AndroidKeyStore
- [ ] **Memory Safety**: Sensitive data cleared from memory when not needed

---

## 6. Performance

### UI Responsiveness
- [ ] **Scroll Performance**: Lists scroll smoothly (60fps)
- [ ] **Animation**: Transitions animate without jank
- [ ] **Memory**: No memory leaks during normal usage

### Loading States
- [ ] **Initial Load**: Loading indicators appear while data loads
- [ ] **Error States**: Error messages display when operations fail
- [ ] **Empty States**: Appropriate UI when no data available

---

## 7. Accessibility

### Screen Reader
- [ ] **Content Descriptions**: All interactive elements have content descriptions
- [ ] **Live Regions**: Status updates announced to screen readers
- [ ] **Focus Navigation**: Keyboard/switch navigation works

### Visual
- [ ] **Color Contrast**: Text meets WCAG AA contrast ratios
- [ ] **Touch Targets**: All touch targets at least 48x48dp
- [ ] **Font Scaling**: UI adapts to font size changes

---

## 8. Edge Cases

### Error Handling
- [ ] **Network Error**: Graceful handling when offline
- [ ] **Invalid Input**: Validation errors show helpful messages
- [ ] **Permission Denied**: Appropriate fallback when permissions denied

### Data Scenarios
- [ ] **Empty Data**: Lists show empty state UI
- [ ] **Large Data**: Long lists scroll and perform well
- [ ] **Corrupt Data**: App doesn't crash on malformed data

---

## Sign-off

**Tester**: _______________
**Date**: _______________
**Build Version**: _______________
**Device**: _______________

---

## Notes

_Use this space for any additional observations or issues found during testing._
