// Package rotate provides generic business logic for rotating passwords. It
// provides methods for talking to rotation and storage client plugins. It will
// go through the configured list of secrets and ask the associated rotation
// client the last time that secret was rotated, the storage clients the last
// time the secret had been stored, apply the rotation policy to determine
// whether the secret needs to be rotated, ask the rotation client to perform
// rotation, and then store the rotated values using each of the configured
// storage clients.
package rotate
