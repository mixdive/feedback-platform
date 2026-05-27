import toast from 'react-hot-toast'

export type MessageKind = 'success' | 'error' | 'info'

export function message(input: unknown, kind: MessageKind = 'error') {
  let text = ''
  if (typeof input === 'string') text = input
  else if (input && typeof input === 'object' && 'message' in input)
    text = String((input as { message: unknown }).message)
  else text = String(input)
  if (kind === 'success') toast.success(text)
  else if (kind === 'error') toast.error(text)
  else toast(text)
}
