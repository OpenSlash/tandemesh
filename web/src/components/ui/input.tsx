import * as React from 'react'
import { cn } from '../../lib/utils'

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        'h-8 w-full rounded-md border border-line2 bg-panel px-2.5 text-[13px] text-fg placeholder:text-muted/70',
        'focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent/40',
        className,
      )}
      {...props}
    />
  ),
)
Input.displayName = 'Input'
