/**
 * Enrollment Store
 *
 * Session-scoped tracking of enrolled device IDs.
 * Not persisted across app restarts — enrolled devices will appear
 * as 'offline' or 'unavailable' after restart since the service is down.
 */

const enrolledDevices = new Map<string, Date>();

export function markEnrolled(deviceId: string): void {
  enrolledDevices.set(deviceId, new Date());
}

export function isEnrolled(deviceId: string): boolean {
  return enrolledDevices.has(deviceId);
}

export function getEnrolledAt(deviceId: string): Date | undefined {
  return enrolledDevices.get(deviceId);
}

export function clearEnrolled(deviceId: string): void {
  enrolledDevices.delete(deviceId);
}

export function getAllEnrolledIds(): Set<string> {
  return new Set(enrolledDevices.keys());
}
