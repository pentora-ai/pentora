import { UserAuthForm } from './components/user-auth-form'

export function SignIn2() {
  return (
    <div className='relative container grid h-svh flex-col items-center justify-center lg:max-w-none lg:grid-cols-2 lg:px-0'>
      <div className='bg-muted relative hidden h-full flex-col p-10 text-white lg:flex dark:border-e'>
        <div className='absolute inset-0 bg-zinc-900' />
        <div className='relative z-20 flex items-center text-lg font-medium'>
          <span>MyCompany</span>
        </div>

        <div className='relative z-20 mt-auto'>
          <blockquote className='space-y-2'>
            <p className='text-lg'></p>
            <footer className='text-sm'></footer>
          </blockquote>
        </div>
      </div>
      <div className='lg:p-8'>
        <div className='mx-auto flex w-full flex-col justify-center space-y-2 sm:w-[350px]'>
          <div className='flex flex-col space-y-2 text-start'>
            <h1 className='text-2xl font-semibold tracking-tight'>Sign in</h1>
            <p className='text-muted-foreground text-sm'>
              Enter your email and password below <br />
              to log into your account
            </p>
          </div>
          <UserAuthForm />
        </div>
      </div>
    </div>
  )
}
