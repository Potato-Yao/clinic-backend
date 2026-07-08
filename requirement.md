# project requirement

## background knowledge

This project is the backend service for a campus computer-clinic appointment system. Its job is to coordinate repair appointments between students who need computer help and the student-staff volunteers who run the clinic. Everything it does serves that single purpose: making sure the right student shows up at the right campus location on the right day, and that the staff can track, process, and close out each repair case.

## technology stack

This project should use go with framework gin to build the project. Use PostgreSQL as database, use go:embed for communicating.

## project's role in detail

- Online repair appointment booking — Students can view available campuses and service dates, then submit a repair ticket with their device info, problem description, and preferred appointment date.
- Staff ticket management — Authorized staff can view, filter, confirm, reject, process, and close repair records through a web admin panel, tracking each case from submission to resolution.
- Service schedule and capacity control — Admins can create service dates per campus with time windows and capacity limits; the system prevents overbooking and blocks changes once records exist.
- Multi-campus support — The system supports multiple clinic locations, each with its own address, service dates, and independent appointment queues.
- Announcement publishing — Staff can post announcements and terms of service with priorities and expiration dates, which client apps can fetch and display.
- Dual authentication — Staff log in through the university CAS system, while customer apps authenticate via a shared API-key signature without needing passwords.
- Nightly automated cleanup — A Celery task runs daily to mark unfinished no-show appointments and auto-close still-in-progress cases from previous days.
- Admin interface — Superusers can directly manage users, records, dates, campuses, and announcements through Django's built-in admin site.
