import React, {useEffect, useRef} from 'react';
import Layout from '@theme-original/Layout';
import {useLocation} from '@docusaurus/router';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';

const MIN_PROGRESS_MS = 300;

NProgress.configure({
  showSpinner: false,
  trickleSpeed: 120,
});

type LayoutProps = React.ComponentProps<typeof Layout>;

export default function LayoutWrapper(props: LayoutProps): React.ReactElement {
  const location = useLocation();
  const firstRenderRef = useRef(true);

  useEffect(() => {
    if (firstRenderRef.current) {
      firstRenderRef.current = false;
      NProgress.done();
      return;
    }

    NProgress.start();
    const timer = window.setTimeout(() => {
      NProgress.done();
    }, MIN_PROGRESS_MS);

    return () => {
      window.clearTimeout(timer);
      NProgress.done();
    };
  }, [location.pathname]);

  return <Layout {...props} />;
}
