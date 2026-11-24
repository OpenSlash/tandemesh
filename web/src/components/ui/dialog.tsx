import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { cn } from '../../lib/utils'

export const Dialog = DialogPrimitive.Root
export const DialogTrigger = DialogPrimitive.Trigger

export function DialogContent({
  className,
  children,
  title,
  description,
}: {
  className?: string
  children: React.ReactNode
  title: string
  description?: string
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/60 backdrop-blur-[1px]" />
      <DialogPrimitive.Content
        className={cn(
          'fixed left-1/2 top-1/2 z-50 w-[440px] -translate-x-1/2 -translate-y-1/2',
          'rounded-lg border border-line2 bg-panel p-5 shadow-2xl focus:outline-none',
          className,
        )}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <DialogPrimitive.Title className="text-[14px] font-medium text-fg">
              {title}
            </DialogPrimitive.Title>
            {description && (
              <DialogPrimitive.Description className="mt-1 text-[12px] text-muted">
                {description}
              </DialogPrimitive.Description>
            )}
          </div>
          <DialogPrimitive.Close className="text-muted hover:text-fg">
            <X className="h-4 w-4" />
          </DialogPrimitive.Close>
        </div>
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  )
}
