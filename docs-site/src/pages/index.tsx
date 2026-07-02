import {useEffect} from 'react';
import {useLocation} from '@docusaurus/router';

export default function Home(): null {
  const location = useLocation();

  useEffect(() => {
    // Redirect /doc/ to /doc/overview (first doc page)
    window.location.href = '/doc/overview';
  }, [location]);

  return null;
}
