/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Loader2, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { MultiKeyConfirmAction, MultiKeyTestResult } from '../../types'

type MultiKeyTableRowActionsProps = {
  keyIndex: number
  status: number
  canDelete: boolean
  testResult?: MultiKeyTestResult
  onAction: (action: MultiKeyConfirmAction) => void
  onTest: (keyIndex: number) => void
}

export function MultiKeyTableRowActions({
  keyIndex,
  status,
  canDelete,
  testResult,
  onAction,
  onTest,
}: MultiKeyTableRowActionsProps) {
  const { t } = useTranslation()
  const isEnabled = status === 1
  const isTesting = testResult?.status === 'testing'

  return (
    <div className='flex items-center justify-end gap-2'>
      <Button
        variant='outline'
        size='sm'
        onClick={() => onTest(keyIndex)}
        disabled={isTesting}
        title={t('Test this key')}
      >
        {isTesting ? (
          <Loader2 className='mr-1 h-3.5 w-3.5 animate-spin' />
        ) : (
          <Zap className='mr-1 h-3.5 w-3.5' />
        )}
        {t('Test')}
      </Button>
      {isEnabled ? (
        <Button
          variant='outline'
          size='sm'
          onClick={() => onAction({ type: 'disable', keyIndex })}
        >
          {t('Disable')}
        </Button>
      ) : (
        <Button
          variant='outline'
          size='sm'
          onClick={() => onAction({ type: 'enable', keyIndex })}
        >
          {t('Enable')}
        </Button>
      )}
      <Button
        variant='destructive'
        size='sm'
        onClick={() => {
          if (!canDelete) return
          onAction({ type: 'delete', keyIndex })
        }}
        disabled={!canDelete}
        title={
          canDelete ? undefined : t('No permission to perform this action')
        }
      >
        {t('Delete')}
      </Button>
    </div>
  )
}
