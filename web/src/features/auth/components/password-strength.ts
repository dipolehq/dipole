/**
 * Password strength scoring and validation.
 * Rules: min 8 chars, lowercase + uppercase, digit, special character.
 * Score 0-4 maps to the robot's meter levels.
 */

export interface PasswordStrength {
  score: number       // 0-4
  label: string       // Display label for the meter
  isValid: boolean    // Meets minimum requirements for submission
  feedback: string    // Human-readable feedback
}

export function evaluatePassword(password: string): PasswordStrength {
  if (!password) {
    return { score: 0, label: 'NOT LOOKING', isValid: false, feedback: '' }
  }

  let score = 0
  const feedback: string[] = []

  if (password.length >= 8) {
    score++
  } else {
    feedback.push('at least 8 characters')
  }

  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) {
    score++
  } else {
    feedback.push('uppercase and lowercase letters')
  }

  if (/\d/.test(password)) {
    score++
  } else {
    feedback.push('a number')
  }

  if (/[^a-zA-Z0-9]/.test(password)) {
    score++
  } else {
    feedback.push('a special character')
  }

  // Ensure at least score 1 if they've typed anything
  if (password.length > 0 && score === 0) score = 1

  const labels = ['NOT LOOKING', 'TOO SHORT', 'GETTING THERE', 'STRONG', 'FORT KNOX']
  const label = labels[score] ?? 'NOT LOOKING'

  // Valid = at least score 3 (8+ chars, mixed case, number OR special)
  const isValid = score >= 3

  const feedbackMsg = feedback.length > 0 ? `Needs ${feedback.join(', ')}` : ''

  return { score, label, isValid, feedback: feedbackMsg }
}
